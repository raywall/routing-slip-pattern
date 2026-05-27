package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/raywall/routing-slip-pattern/slip"
	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	Service       ServiceConfig       `yaml:"service"`
	Trigger       TriggerConfig       `yaml:"trigger"`
	Features      FeatureFlagsConfig  `yaml:"features"`
	Metrics       MetricsConfig       `yaml:"metrics"`
	Observability ObservabilityConfig `yaml:"observability"`
	Security      SecurityConfig      `yaml:"security"`
	Integrations  IntegrationsConfig  `yaml:"integrations"`
}

type ServiceConfig struct {
	Name  string `yaml:"name"`
	RunID string `yaml:"run_id"`
}

type TriggerConfig struct {
	Type  string             `yaml:"type"`
	REST  RESTTriggerConfig  `yaml:"rest"`
	Kafka KafkaTriggerConfig `yaml:"kafka"`
	SQS   SQSTriggerConfig   `yaml:"sqs"`
}

type RESTTriggerConfig struct {
	Addr string `yaml:"addr"`
	Path string `yaml:"path"`
}

type KafkaTriggerConfig struct {
	Brokers  []string `yaml:"brokers"`
	Topic    string   `yaml:"topic"`
	GroupID  string   `yaml:"group_id"`
	MinBytes int      `yaml:"min_bytes"`
	MaxBytes int      `yaml:"max_bytes"`
}

type SQSTriggerConfig struct {
	QueueURL          string `yaml:"queue_url"`
	Endpoint          string `yaml:"endpoint"`
	Region            string `yaml:"region"`
	WaitTimeSeconds   int32  `yaml:"wait_time_seconds"`
	MaxMessages       int32  `yaml:"max_messages"`
	VisibilityTimeout int32  `yaml:"visibility_timeout"`
}

type WorkflowConfig struct {
	Name              string       `yaml:"name"`
	Description       string       `yaml:"description"`
	ErrorPolicy       string       `yaml:"error_policy"`
	MessageIDPath     string       `yaml:"message_id_path"`
	CorrelationIDPath string       `yaml:"correlation_id_path"`
	Steps             []StepConfig `yaml:"steps"`
}

type StepConfig struct {
	ID      string         `yaml:"id"`
	Name    string         `yaml:"name"`
	Enabled *bool          `yaml:"enabled"`
	Params  map[string]any `yaml:"params"`
}

type MetricsConfig struct {
	Endpoint string            `yaml:"endpoint"`
	Segment  string            `yaml:"segment"`
	Source   string            `yaml:"source"`
	Tags     map[string]string `yaml:"tags"`
}

type FeatureFlagsConfig struct {
	TracingEnabled         *bool `yaml:"tracing_enabled"`
	MCPEnabled             bool  `yaml:"mcp_enabled"`
	AsyncMetricsEnabled    bool  `yaml:"async_metrics_enabled"`
	PersistentStateEnabled bool  `yaml:"persistent_state_enabled"`
}

type ObservabilityConfig struct {
	Tracing TracingConfig `yaml:"tracing"`
}

type TracingConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Exporter    string `yaml:"exporter"`
	Endpoint    string `yaml:"endpoint"`
	ServiceName string `yaml:"service_name"`
}

type SecurityConfig struct {
	Redaction RedactionConfig `yaml:"redaction"`
}

type RedactionConfig struct {
	Enabled bool     `yaml:"enabled"`
	Fields  []string `yaml:"fields"`
}

type IntegrationsConfig struct {
	GraphQLEndpoint   string `yaml:"graphql_endpoint"`
	ExternalAPIURL    string `yaml:"external_api_url"`
	ExternalAPISerial string `yaml:"external_api_serial"`
}

func loadAppConfig(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config %q: %w", path, err)
	}

	expanded := expandEnvDefaults(string(data))
	var cfg AppConfig
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("invalid config %q: %w", path, err)
	}
	applyConfigDefaults(&cfg)
	if err := validateAppConfig(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func loadWorkflowConfig(path string) (*WorkflowConfig, error) {
	return loadWorkflowConfigWithStack(path, map[string]bool{})
}

func loadWorkflowConfigWithStack(path string, stack map[string]bool) (*WorkflowConfig, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve workflow %q: %w", path, err)
	}
	if stack[absPath] {
		return nil, fmt.Errorf("workflow reference cycle detected at %q", path)
	}
	stack[absPath] = true
	defer delete(stack, absPath)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read workflow %q: %w", path, err)
	}

	expanded := expandEnvDefaults(string(data))
	var workflow WorkflowConfig
	if err := yaml.Unmarshal([]byte(expanded), &workflow); err != nil {
		return nil, fmt.Errorf("invalid workflow %q: %w", path, err)
	}
	applyWorkflowDefaults(&workflow)
	if err := validateWorkflowConfig(&workflow); err != nil {
		return nil, err
	}
	if err := expandWorkflowReferences(&workflow, absPath, stack); err != nil {
		return nil, err
	}
	return &workflow, nil
}

func expandWorkflowReferences(workflow *WorkflowConfig, sourcePath string, stack map[string]bool) error {
	expanded := make([]StepConfig, 0, len(workflow.Steps))
	for _, step := range workflow.Steps {
		if step.Enabled != nil && !*step.Enabled {
			expanded = append(expanded, step)
			continue
		}
		if step.Name != "workflow_ref" {
			expanded = append(expanded, step)
			continue
		}
		refPath, err := workflowRefPath(step)
		if err != nil {
			return err
		}
		if !filepath.IsAbs(refPath) {
			refPath = filepath.Join(filepath.Dir(sourcePath), refPath)
		}
		child, err := loadWorkflowConfigWithStack(refPath, stack)
		if err != nil {
			return fmt.Errorf("workflow_ref %q: %w", refPath, err)
		}
		prefix := workflowRefPrefix(step, child, refPath)
		childIDs := map[string]bool{}
		for _, childStep := range child.Steps {
			if strings.TrimSpace(childStep.ID) != "" {
				childIDs[childStep.ID] = true
			}
		}
		for index, childStep := range child.Steps {
			childStep.ID = prefixedStepID(prefix, childStep.ID, childStep.Name, index)
			childStep.Params = rewriteWorkflowRefTargets(copyParams(childStep.Params), prefix, childIDs)
			expanded = append(expanded, childStep)
		}
	}
	workflow.Steps = expanded
	return nil
}

func workflowRefPath(step StepConfig) (string, error) {
	if step.Params == nil {
		return "", fmt.Errorf("workflow_ref %q requires params.file, params.path or params.workflow", step.ID)
	}
	for _, key := range []string{"file", "path", "workflow"} {
		if value, ok := step.Params[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value), nil
		}
	}
	return "", fmt.Errorf("workflow_ref %q requires params.file, params.path or params.workflow", step.ID)
}

func workflowRefPrefix(step StepConfig, child *WorkflowConfig, refPath string) string {
	for _, value := range []string{
		stringParam(step.Params, "prefix"),
		step.ID,
		child.Name,
		strings.TrimSuffix(filepath.Base(refPath), filepath.Ext(refPath)),
	} {
		if cleaned := cleanStepID(value); cleaned != "" {
			return cleaned
		}
	}
	return "workflow"
}

func prefixedStepID(prefix, id, name string, index int) string {
	if cleaned := cleanStepID(id); cleaned != "" {
		return prefix + "." + cleaned
	}
	if cleaned := cleanStepID(name); cleaned != "" {
		return fmt.Sprintf("%s.%03d.%s", prefix, index+1, cleaned)
	}
	return fmt.Sprintf("%s.%03d", prefix, index+1)
}

func rewriteWorkflowRefTargets(params map[string]any, prefix string, childIDs map[string]bool) map[string]any {
	if params == nil {
		return nil
	}
	for key, value := range params {
		if key == "to" {
			if target, ok := value.(string); ok && childIDs[target] {
				params[key] = prefix + "." + target
			}
			continue
		}
		switch typed := value.(type) {
		case map[string]any:
			params[key] = rewriteWorkflowRefTargets(typed, prefix, childIDs)
		case []any:
			for index, item := range typed {
				if nested, ok := item.(map[string]any); ok {
					typed[index] = rewriteWorkflowRefTargets(nested, prefix, childIDs)
				}
			}
			params[key] = typed
		}
	}
	return params
}

func copyParams(params map[string]any) map[string]any {
	if params == nil {
		return nil
	}
	copied := make(map[string]any, len(params))
	for key, value := range params {
		copied[key] = copyAny(value)
	}
	return copied
}

func copyAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return copyParams(typed)
	case []any:
		copied := make([]any, len(typed))
		for index, item := range typed {
			copied[index] = copyAny(item)
		}
		return copied
	default:
		return value
	}
}

func stringParam(params map[string]any, key string) string {
	if params == nil {
		return ""
	}
	value, _ := params[key].(string)
	return value
}

func cleanStepID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var out strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '.' {
			out.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			out.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(out.String(), "-.")
}

func expandEnvDefaults(input string) string {
	var out strings.Builder
	for i := 0; i < len(input); i++ {
		if input[i] != '$' || i+1 >= len(input) || input[i+1] != '{' {
			out.WriteByte(input[i])
			continue
		}

		end := strings.IndexByte(input[i+2:], '}')
		if end < 0 {
			out.WriteByte(input[i])
			continue
		}

		expr := input[i+2 : i+2+end]
		out.WriteString(expandEnvExpression(expr))
		i += end + 2
	}
	return out.String()
}

func expandEnvExpression(key string) string {
	if name, fallback, ok := strings.Cut(key, ":-"); ok {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
		return fallback
	}
	return os.Getenv(key)
}

func expandEnvDefaultsLegacy(input string) string {
	return os.Expand(input, func(key string) string {
		if name, fallback, ok := strings.Cut(key, ":-"); ok {
			if value := strings.TrimSpace(os.Getenv(name)); value != "" {
				return value
			}
			return fallback
		}
		return os.Getenv(key)
	})
}

func applyConfigDefaults(cfg *AppConfig) {
	if strings.TrimSpace(cfg.Service.Name) == "" {
		cfg.Service.Name = "routing-slip-pattern"
	}
	if strings.TrimSpace(cfg.Service.RunID) == "" {
		cfg.Service.RunID = time.Now().Format("20060102150405")
	}
	if strings.TrimSpace(cfg.Trigger.Type) == "" {
		cfg.Trigger.Type = "rest"
	}
	cfg.Trigger.Type = strings.ToLower(strings.TrimSpace(cfg.Trigger.Type))
	if strings.TrimSpace(cfg.Trigger.REST.Addr) == "" {
		cfg.Trigger.REST.Addr = ":8088"
	}
	if strings.TrimSpace(cfg.Trigger.REST.Path) == "" {
		cfg.Trigger.REST.Path = "/process"
	}
	if len(cfg.Trigger.Kafka.Brokers) == 0 {
		cfg.Trigger.Kafka.Brokers = []string{"localhost:9092"}
	}
	if len(cfg.Trigger.Kafka.Brokers) == 1 && strings.Contains(cfg.Trigger.Kafka.Brokers[0], ",") {
		cfg.Trigger.Kafka.Brokers = splitCSV(cfg.Trigger.Kafka.Brokers[0])
	}
	if strings.TrimSpace(cfg.Trigger.Kafka.Topic) == "" {
		cfg.Trigger.Kafka.Topic = "payment-events"
	}
	if strings.TrimSpace(cfg.Trigger.Kafka.GroupID) == "" {
		cfg.Trigger.Kafka.GroupID = "routing-slip-pattern"
	}
	if cfg.Trigger.Kafka.MinBytes <= 0 {
		cfg.Trigger.Kafka.MinBytes = 1
	}
	if cfg.Trigger.Kafka.MaxBytes <= 0 {
		cfg.Trigger.Kafka.MaxBytes = 10e6
	}
	if strings.TrimSpace(cfg.Trigger.SQS.Region) == "" {
		cfg.Trigger.SQS.Region = "us-east-1"
	}
	if strings.TrimSpace(cfg.Trigger.SQS.Endpoint) == "" {
		cfg.Trigger.SQS.Endpoint = "http://localhost:4566"
	}
	if strings.TrimSpace(cfg.Trigger.SQS.QueueURL) == "" {
		cfg.Trigger.SQS.QueueURL = "http://localhost:4566/000000000000/payment-events"
	}
	if cfg.Trigger.SQS.WaitTimeSeconds <= 0 {
		cfg.Trigger.SQS.WaitTimeSeconds = 10
	}
	if cfg.Trigger.SQS.MaxMessages <= 0 {
		cfg.Trigger.SQS.MaxMessages = 1
	}
	if cfg.Trigger.SQS.VisibilityTimeout <= 0 {
		cfg.Trigger.SQS.VisibilityTimeout = 30
	}
	if strings.TrimSpace(cfg.Metrics.Endpoint) == "" {
		cfg.Metrics.Endpoint = env("METRICS_ENDPOINT", "http://localhost:8080/v1/metrics")
	}
	if strings.TrimSpace(cfg.Metrics.Segment) == "" {
		cfg.Metrics.Segment = "local-demo"
	}
	if strings.TrimSpace(cfg.Metrics.Source) == "" {
		cfg.Metrics.Source = "routing-slip-app"
	}
	if cfg.Metrics.Tags == nil {
		cfg.Metrics.Tags = map[string]string{}
	}
	if cfg.Metrics.Tags["run_id"] == "" {
		cfg.Metrics.Tags["run_id"] = cfg.Service.RunID
	}
	if cfg.Features.TracingEnabled == nil {
		enabled := true
		cfg.Features.TracingEnabled = &enabled
	}
	if strings.TrimSpace(cfg.Observability.Tracing.Exporter) == "" {
		cfg.Observability.Tracing.Exporter = "none"
	}
	if strings.TrimSpace(cfg.Observability.Tracing.ServiceName) == "" {
		cfg.Observability.Tracing.ServiceName = cfg.Service.Name
	}
	if !cfg.Security.Redaction.Enabled && len(cfg.Security.Redaction.Fields) == 0 {
		cfg.Security.Redaction.Enabled = true
		cfg.Security.Redaction.Fields = []string{"authorization", "client_secret", "access_token", "refresh_token", "password", "token", "api_key", "x-api-key"}
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func applyWorkflowDefaults(workflow *WorkflowConfig) {
	if strings.TrimSpace(workflow.Name) == "" {
		workflow.Name = "payment-event-fulfillment"
	}
	if strings.TrimSpace(workflow.ErrorPolicy) == "" {
		workflow.ErrorPolicy = "stop"
	}
	if strings.TrimSpace(workflow.MessageIDPath) == "" {
		workflow.MessageIDPath = "payload.pagamento_id"
	}
	if strings.TrimSpace(workflow.CorrelationIDPath) == "" {
		workflow.CorrelationIDPath = "correlation_id"
	}
}

func validateAppConfig(cfg *AppConfig) error {
	switch cfg.Trigger.Type {
	case "rest", "kafka", "sqs":
	default:
		return fmt.Errorf("unsupported trigger type %q: use rest, kafka or sqs", cfg.Trigger.Type)
	}
	return nil
}

func validateWorkflowConfig(workflow *WorkflowConfig) error {
	if len(workflow.Steps) == 0 {
		return fmt.Errorf("workflow %q has no steps", workflow.Name)
	}
	return nil
}

func (workflow WorkflowConfig) ToSlip() []slip.StepDef {
	steps := make([]slip.StepDef, 0, len(workflow.Steps))
	for _, step := range workflow.Steps {
		if step.Enabled != nil && !*step.Enabled {
			continue
		}
		steps = append(steps, slip.StepDef{ID: step.ID, Name: step.Name, Params: step.Params})
	}
	return steps
}

func (workflow WorkflowConfig) RoutingErrorPolicy() (slip.ErrorPolicy, error) {
	switch strings.ToLower(strings.TrimSpace(workflow.ErrorPolicy)) {
	case "", "stop":
		return slip.StopOnError, nil
	case "continue":
		return slip.ContinueOnError, nil
	case "skip":
		return slip.SkipOnError, nil
	default:
		return slip.StopOnError, fmt.Errorf("unknown error policy %q", workflow.ErrorPolicy)
	}
}

func (cfg AppConfig) ApplyIntegrationEnv() {
	setEnvIfEmpty("GRAPHQL_ENDPOINT", cfg.Integrations.GraphQLEndpoint)
	setEnvIfEmpty("EXTERNAL_API_URL", cfg.Integrations.ExternalAPIURL)
	setEnvIfEmpty("EXTERNAL_API_SERIAL", cfg.Integrations.ExternalAPISerial)
	if cfg.Features.TracingEnabled != nil {
		setEnvBoolIfEmpty("ROUTING_SLIP_TRACING_ENABLED", *cfg.Features.TracingEnabled)
	}
}

func setEnvIfEmpty(key, value string) {
	value = strings.TrimSpace(value)
	if value == "" || strings.TrimSpace(os.Getenv(key)) != "" {
		return
	}
	_ = os.Setenv(key, value)
}

func setEnvBoolIfEmpty(key string, value bool) {
	if strings.TrimSpace(os.Getenv(key)) != "" {
		return
	}
	if value {
		_ = os.Setenv(key, "true")
		return
	}
	_ = os.Setenv(key, "false")
}

func stringFromPath(values map[string]any, path string) (string, bool) {
	value, ok := valueFromPath(values, path)
	if !ok || value == nil {
		return "", false
	}
	switch typed := value.(type) {
	case string:
		return typed, typed != ""
	case fmt.Stringer:
		return typed.String(), true
	case int:
		return strconv.Itoa(typed), true
	case int64:
		return strconv.FormatInt(typed, 10), true
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(typed), true
	default:
		return fmt.Sprint(typed), true
	}
}

func valueFromPath(values map[string]any, path string) (any, bool) {
	if strings.TrimSpace(path) == "" {
		return nil, false
	}
	var current any = values
	for _, part := range strings.Split(path, ".") {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}
