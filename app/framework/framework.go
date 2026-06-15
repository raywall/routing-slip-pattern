// Package framework provides the importable routing-slip runtime.
package framework

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/raywall/routing-slip-pattern/app/handlers"
	"github.com/raywall/routing-slip-pattern/app/slip"
	"github.com/raywall/routing-slip-pattern/app/source"
	"gopkg.in/yaml.v3"
)

// MetricsAgent is implemented by custom-business-metrics/agent.Agent.
type MetricsAgent interface {
	Emit(context.Context, any) error
}

// Config controls the embeddable runtime.
type Config struct {
	Service struct {
		Name  string `yaml:"name"`
		RunID string `yaml:"run_id"`
	} `yaml:"service"`
	REST struct {
		Addr string `yaml:"addr"`
		Path string `yaml:"path"`
	} `yaml:"rest"`
	Trigger struct {
		Connector string `yaml:"connector"`
		Mode      string `yaml:"mode"`
		REST      struct {
			Addr string `yaml:"addr"`
			Path string `yaml:"path"`
		} `yaml:"rest"`
	} `yaml:"trigger"`
	MCP struct {
		Enabled   bool   `yaml:"enabled"`
		Addr      string `yaml:"addr"`
		Bind      string `yaml:"bind"`
		APIKey    string `yaml:"api_key"`
		APIKeyEnv string `yaml:"api_key_env"`
	} `yaml:"mcp"`
	StateStore struct {
		Type        string `yaml:"type"`
		Path        string `yaml:"path"`
		Table       string `yaml:"table"`
		Region      string `yaml:"region"`
		Endpoint    string `yaml:"endpoint"`
		TTLDays     int    `yaml:"ttl_days"`
		Idempotency struct {
			Enabled     bool   `yaml:"enabled"`
			KeyTemplate string `yaml:"key_template"`
		} `yaml:"idempotency"`
	} `yaml:"state_store"`
	Idempotency struct {
		Enabled     bool   `yaml:"enabled"`
		KeyTemplate string `yaml:"key_template"`
	} `yaml:"idempotency"`
	GraphQLEndpoint string `yaml:"graphql_endpoint"`
	Integrations    struct {
		GraphQLEndpoint string `yaml:"graphql_endpoint"`
	} `yaml:"integrations"`
}

// BusinessRule provides human-readable rule metadata exposed through MCP.
type BusinessRule struct {
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	Status      string `yaml:"status" json:"status"`
}

// Workflow is the serialized workflow contract.
type Workflow struct {
	Name              string         `yaml:"name" json:"name"`
	Description       string         `yaml:"description" json:"description"`
	Version           string         `yaml:"version" json:"version"`
	ErrorPolicy       string         `yaml:"error_policy" json:"error_policy"`
	MessageIDPath     string         `yaml:"message_id_path" json:"message_id_path"`
	CorrelationIDPath string         `yaml:"correlation_id_path" json:"correlation_id_path"`
	BusinessRules     []BusinessRule `yaml:"business_rules,omitempty" json:"business_rules,omitempty"`
	Steps             []Step         `yaml:"steps" json:"steps"`
}

// Step describes a workflow step.
type Step struct {
	ID         string                `yaml:"id" json:"id"`
	Name       string                `yaml:"name" json:"name"`
	Enabled    *bool                 `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Params     map[string]any        `yaml:"params" json:"params"`
	Resilience slip.ResiliencePolicy `yaml:"resilience,omitempty" json:"resilience,omitempty"`
}

// Options defines all dependencies needed to instantiate a runtime.
type Options struct {
	ConfigSource   source.Source
	WorkflowSource source.Source
	MetricsAgent   MetricsAgent
	StateStore     slip.StateStore
	Logger         *slog.Logger
	HTTPClient     *http.Client
}

// Runtime executes a loaded workflow and exposes REST and MCP handlers.
type Runtime struct {
	config   Config
	workflow Workflow
	router   *slip.Router
	store    slip.StateStore
	steps    []slip.StepDef
	logger   *slog.Logger
}

// New loads configuration and workflow from the provided origins.
func New(ctx context.Context, options Options) (*Runtime, error) {
	if options.ConfigSource == nil || options.WorkflowSource == nil {
		return nil, fmt.Errorf("config source and workflow source are required")
	}
	var config Config
	if err := loadYAML(ctx, options.ConfigSource, &config); err != nil {
		return nil, fmt.Errorf("load runtime config: %w", err)
	}
	var workflow Workflow
	if err := loadWorkflow(ctx, options.WorkflowSource, &workflow, map[string]bool{}); err != nil {
		return nil, fmt.Errorf("load workflow: %w", err)
	}
	applyDefaults(&config, &workflow)
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}
	store := options.StateStore
	if store == nil {
		var err error
		store, err = buildStore(ctx, config)
		if err != nil {
			return nil, err
		}
	}
	policy, err := errorPolicy(workflow.ErrorPolicy)
	if err != nil {
		return nil, err
	}
	routerOptions := []slip.RouterOption{
		slip.WithLogger(logger),
		slip.WithErrorPolicy(policy),
		slip.WithStateStore(store),
		slip.WithStateOptions(slip.StateOptions{
			Workflow: workflow.Name, WorkflowVersion: workflow.Version,
			IdempotencyEnabled: config.Idempotency.Enabled, IdempotencyKeyTemplate: config.Idempotency.KeyTemplate,
		}),
		slip.WithMiddleware(slip.RecoveryMiddleware(), slip.LoggingMiddleware(logger), slip.TracingMiddleware()),
	}
	if options.MetricsAgent != nil {
		routerOptions = append(routerOptions, slip.WithMiddleware(slip.MetricsMiddleware(agentEmitter{options.MetricsAgent}, slip.MetricsOptions{
			Workflow: workflow.Name, Source: config.Service.Name, Tags: map[string]string{"run_id": config.Service.RunID},
		}, logger)))
	}
	router := slip.NewRouter(routerOptions...)
	registerHandlers(router, config.GraphQLEndpoint, options.HTTPClient)
	return &Runtime{config: config, workflow: workflow, router: router, store: store, steps: workflow.toSlip(), logger: logger}, nil
}

// Process executes or resumes one payload.
func (r *Runtime) Process(ctx context.Context, payload map[string]any) (*slip.Message, error) {
	correlationID, ok := stringPath(payload, r.workflow.CorrelationIDPath)
	if !ok {
		correlationID = uuid()
		setPath(payload, r.workflow.CorrelationIDPath, correlationID)
	}
	messageID, ok := stringPath(payload, r.workflow.MessageIDPath)
	if !ok {
		messageID = correlationID
	}
	if locker, ok := r.store.(slip.ProcessingLocker); ok {
		lease, acquired, err := locker.TryAcquireProcessing(ctx, messageID, r.config.Service.Name+":"+r.config.Service.RunID, 5*time.Minute)
		if err != nil {
			return nil, err
		}
		if !acquired {
			return nil, fmt.Errorf("%w: %s", slip.ErrProcessingLocked, messageID)
		}
		defer func() { _ = lease.Release(context.Background()) }()
	}
	msg := slip.NewMessage(messageID, payload)
	if snapshot, err := r.store.Load(ctx, messageID); err == nil {
		msg = slip.MessageFromSnapshot(snapshot)
	} else if !slip.IsStateNotFound(err) {
		return nil, err
	}
	if msg.CorrelationID == "" {
		msg.CorrelationID = correlationID
	}
	msg.Headers["correlation_id"] = msg.CorrelationID
	msg.Headers["workflow"] = r.workflow.Name
	if msg.RemainingSteps() == 0 && msg.Cursor() == 0 {
		msg.AttachSlip(r.steps)
	}
	if msg.Status == "completed" && msg.RemainingSteps() == 0 {
		return msg, nil
	}
	return msg, r.router.Process(ctx, msg)
}

// Handler returns a synchronous REST handler suitable for ECS, EKS, Lambda URLs, ALB and API Gateway adapters.
func (r *Runtime) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var payload map[string]any
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		message, err := r.Process(req.Context(), payload)
		status := http.StatusOK
		if err != nil {
			status = http.StatusUnprocessableEntity
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": message, "error": errorText(err)})
	})
}

// ListenAndServe starts the configured REST server.
func (r *Runtime) ListenAndServe(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.Handle(r.config.REST.Path, r.Handler())
	server := &http.Server{Addr: r.config.REST.Addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	err := server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// Run exposes the configured REST API and optional MCP server.
func (r *Runtime) Run(ctx context.Context) error {
	if !r.config.MCP.Enabled {
		return r.ListenAndServe(ctx)
	}
	mcp := &http.Server{Addr: r.config.MCP.Addr, Handler: r.authorizedMCP(), ReadHeaderTimeout: 5 * time.Second}
	errs := make(chan error, 2)
	go func() { errs <- r.ListenAndServe(ctx) }()
	go func() {
		err := mcp.ListenAndServe()
		if err == http.ErrServerClosed {
			err = nil
		}
		errs <- err
	}()
	select {
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mcp.Shutdown(shutdown)
		return ctx.Err()
	case err := <-errs:
		return err
	}
}

// MCPHandler exposes workflows, business rules and execution snapshots to LLM clients.
func (r *Runtime) MCPHandler() http.Handler { return newMCPHandler(r.workflow, r.store) }

func (r *Runtime) authorizedMCP() http.Handler {
	next := r.MCPHandler()
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if r.config.MCP.APIKey != "" && req.Header.Get("X-API-Key") != r.config.MCP.APIKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, req)
	})
}

func loadYAML(ctx context.Context, src source.Source, target any) error {
	data, err := src.Load(ctx)
	if err != nil {
		return err
	}
	return yaml.Unmarshal([]byte(expandEnvironment(string(data))), target)
}

func loadWorkflow(ctx context.Context, src source.Source, workflow *Workflow, stack map[string]bool) error {
	data, err := src.Load(ctx)
	if err != nil {
		return err
	}
	var project struct {
		WorkflowScript *Workflow      `yaml:"workflow_script"`
		BusinessRules  []BusinessRule `yaml:"business_rules"`
	}
	data = []byte(expandEnvironment(string(data)))
	if err := yaml.Unmarshal(data, &project); err != nil {
		return err
	}
	if project.WorkflowScript != nil {
		*workflow = *project.WorkflowScript
		if len(workflow.BusinessRules) == 0 {
			workflow.BusinessRules = project.BusinessRules
		}
	} else if err := yaml.Unmarshal(data, workflow); err != nil {
		return err
	}
	resolver, _ := src.(source.Resolver)
	expanded := make([]Step, 0, len(workflow.Steps))
	for _, step := range workflow.Steps {
		if step.Name != "workflow_ref" {
			expanded = append(expanded, step)
			continue
		}
		ref := firstValue(step.Params, "path", "workflow", "ref")
		if resolver == nil || ref == "" {
			return fmt.Errorf("workflow_ref %q requires a resolvable source", ref)
		}
		if stack[ref] {
			return fmt.Errorf("workflow reference cycle at %q", ref)
		}
		stack[ref] = true
		childSource, err := resolver.Resolve(ref)
		if err != nil {
			return err
		}
		var child Workflow
		if err := loadWorkflow(ctx, childSource, &child, stack); err != nil {
			return err
		}
		delete(stack, ref)
		expanded = append(expanded, child.Steps...)
	}
	workflow.Steps = expanded
	return nil
}

func applyDefaults(config *Config, workflow *Workflow) {
	if config.Service.Name == "" {
		config.Service.Name = "routing-slip-pattern"
	}
	if config.Service.RunID == "" {
		config.Service.RunID = uuid()
	}
	if config.REST.Addr == "" {
		config.REST.Addr = config.Trigger.REST.Addr
	}
	if config.REST.Path == "" {
		config.REST.Path = config.Trigger.REST.Path
	}
	if config.REST.Addr == "" {
		config.REST.Addr = ":8088"
	}
	if config.REST.Path == "" {
		config.REST.Path = "/process"
	}
	if config.MCP.Addr == "" {
		config.MCP.Addr = config.MCP.Bind
	}
	if config.MCP.Addr == "" {
		config.MCP.Addr = ":9091"
	}
	if config.MCP.APIKey == "" && config.MCP.APIKeyEnv != "" {
		config.MCP.APIKey = os.Getenv(config.MCP.APIKeyEnv)
	}
	if config.GraphQLEndpoint == "" {
		config.GraphQLEndpoint = config.Integrations.GraphQLEndpoint
	}
	if !config.Idempotency.Enabled && config.StateStore.Idempotency.Enabled {
		config.Idempotency.Enabled = true
	}
	if config.Idempotency.KeyTemplate == "" {
		config.Idempotency.KeyTemplate = config.StateStore.Idempotency.KeyTemplate
	}
	if workflow.ErrorPolicy == "" {
		workflow.ErrorPolicy = "stop"
	}
	if workflow.MessageIDPath == "" {
		workflow.MessageIDPath = "message_id"
	}
	if workflow.CorrelationIDPath == "" {
		workflow.CorrelationIDPath = "correlation_id"
	}
	if config.Idempotency.KeyTemplate == "" {
		config.Idempotency.KeyTemplate = "{workflow}:{message_id}:{step_index}:{step}"
	}
}

func buildStore(ctx context.Context, config Config) (slip.StateStore, error) {
	switch strings.ToLower(config.StateStore.Type) {
	case "", "memory":
		return slip.NewMemoryStateStore(), nil
	case "file":
		return slip.NewFileStateStore(config.StateStore.Path)
	case "dynamodb":
		options := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(config.StateStore.Region)}
		if config.StateStore.Endpoint != "" {
			options = append(options,
				awsconfig.WithBaseEndpoint(config.StateStore.Endpoint),
				awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("local", "local", "")),
			)
		}
		cfg, err := awsconfig.LoadDefaultConfig(ctx, options...)
		if err != nil {
			return nil, err
		}
		return slip.NewDynamoDBStateStore(dynamodb.NewFromConfig(cfg), config.StateStore.Table, config.StateStore.TTLDays)
	default:
		return nil, fmt.Errorf("unsupported state store %q", config.StateStore.Type)
	}
}

func registerHandlers(router *slip.Router, graphqlEndpoint string, client *http.Client) {
	router.MustRegister(handlers.ValidationHandler{})
	router.MustRegister(handlers.EnrichmentHandler{})
	router.MustRegister(handlers.AssertHandler{})
	router.MustRegister(handlers.ComputeHandler{})
	router.MustRegister(handlers.CELHandler{})
	router.MustRegister(handlers.FilterArrayHandler{})
	router.MustRegister(handlers.ArrayTransformHandler{})
	router.MustRegister(handlers.TransformHandler{})
	router.MustRegister(handlers.ConditionGate{})
	router.MustRegister(handlers.JumpIfHandler{})
	router.MustRegister(&handlers.NotificationHandler{})
	router.MustRegister(handlers.LogHandler{})
	router.MustRegister(handlers.AuditHandler{})
	router.MustRegister(handlers.DatadogMetricHandler{})
	router.MustRegister(handlers.GraphQLEnrichmentHandler{DefaultEndpoint: graphqlEndpoint, Client: client})
	router.MustRegister(handlers.RESTCallHandler{Client: client})
	router.MustRegister(handlers.AWSActionHandler{})
}

func (w Workflow) toSlip() []slip.StepDef {
	out := make([]slip.StepDef, 0, len(w.Steps))
	for _, step := range w.Steps {
		if step.Enabled != nil && !*step.Enabled {
			continue
		}
		out = append(out, slip.StepDef{ID: step.ID, Name: step.Name, Params: step.Params, Resilience: step.Resilience})
	}
	return out
}

type agentEmitter struct{ agent MetricsAgent }

func (e agentEmitter) Emit(ctx context.Context, event slip.MetricEvent) error {
	return e.agent.Emit(ctx, event)
}

func errorPolicy(value string) (slip.ErrorPolicy, error) {
	switch strings.ToLower(value) {
	case "", "stop":
		return slip.StopOnError, nil
	case "continue":
		return slip.ContinueOnError, nil
	case "skip":
		return slip.SkipOnError, nil
	default:
		return slip.StopOnError, fmt.Errorf("unsupported error policy %q", value)
	}
}

func uuid() string {
	var value [16]byte
	_, _ = rand.Read(value[:])
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	text := hex.EncodeToString(value[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", text[:8], text[8:12], text[12:16], text[16:20], text[20:])
}

func stringPath(payload map[string]any, path string) (string, bool) {
	current := any(payload)
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return "", false
		}
		current, ok = object[part]
		if !ok {
			return "", false
		}
	}
	value := strings.TrimSpace(fmt.Sprint(current))
	return value, value != ""
}

func setPath(payload map[string]any, path string, value any) {
	parts := strings.Split(path, ".")
	current := payload
	for _, part := range parts[:len(parts)-1] {
		next, _ := current[part].(map[string]any)
		if next == nil {
			next = map[string]any{}
			current[part] = next
		}
		current = next
	}
	current[parts[len(parts)-1]] = value
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func firstValue(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(fmt.Sprint(values[key])); value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}
