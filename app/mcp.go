package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/raywall/routing-slip-pattern/slip"
	"gopkg.in/yaml.v3"
)

type mcpServer struct {
	cfg      *AppConfig
	workflow *WorkflowConfig
	store    slip.StateStore
	logger   *slog.Logger
	tools    map[string]mcpTool
}

type mcpTool struct {
	Name        string                                             `json:"name"`
	Description string                                             `json:"description"`
	ReadOnly    bool                                               `json:"readOnly"`
	InputSchema map[string]any                                     `json:"inputSchema"`
	Handler     func(context.Context, map[string]any) (any, error) `json:"-"`
}

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type mcpCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func newMCPServer(cfg *AppConfig, workflow *WorkflowConfig, store slip.StateStore, logger *slog.Logger) *mcpServer {
	server := &mcpServer{cfg: cfg, workflow: workflow, store: store, logger: logger}
	server.tools = map[string]mcpTool{}
	server.registerTools()
	return server
}

func (s *mcpServer) registerTools() {
	for _, tool := range []mcpTool{
		{Name: "list_handlers", Description: "Lista handlers registrados e parametros esperados.", ReadOnly: true, Handler: s.listHandlers},
		{Name: "validate_workflow", Description: "Valida YAML, handlers conhecidos, saltos e composicao.", ReadOnly: true, Handler: s.validateWorkflow},
		{Name: "explain_workflow", Description: "Explica fluxo, decisoes, integracoes e pontos de parada.", ReadOnly: true, Handler: s.explainWorkflow},
		{Name: "export_workflow", Description: "Une workflows referenciados em um unico arquivo YAML.", ReadOnly: true, Handler: s.exportWorkflow},
		{Name: "get_execution", Description: "Recupera execucao por message_id, correlation_id ou trace_id.", ReadOnly: true, Handler: s.getExecution},
		{Name: "list_state_snapshots", Description: "Lista snapshots persistidos por filtro simples.", ReadOnly: true, Handler: s.listStateSnapshots},
		{Name: "plan_workflow", Description: "Gera rascunho de workflow a partir de descricao, evento e integracoes.", ReadOnly: true, Handler: s.planWorkflow},
		{Name: "generate_workflow_from_business_rules", Description: "Gera workflow YAML e payload base a partir de regras de negocio.", ReadOnly: true, Handler: s.generateWorkflowFromBusinessRules},
		{Name: "validate_workflow_against_business_rules", Description: "Valida se o workflow cobre as regras de negocio ativas informadas.", ReadOnly: true, Handler: s.validateWorkflowAgainstBusinessRules},
		{Name: "suggest_handlers", Description: "Sugere handlers adequados para capacidades desejadas.", ReadOnly: true, Handler: s.suggestHandlers},
		{Name: "generate_test_payload", Description: "Gera payload de teste a partir do workflow ou descricao.", ReadOnly: true, Handler: s.generateTestPayload},
		{Name: "generate_bruno_collection", Description: "Gera modelo textual de requisicoes Bruno para REST e MCP.", ReadOnly: true, Handler: s.generateBrunoCollection},
		{Name: "assess_idempotency", Description: "Analisa riscos de idempotencia e side effects do workflow.", ReadOnly: true, Handler: s.assessIdempotency},
		{Name: "suggest_metrics", Description: "Sugere metricas e pontos de auditoria para o workflow.", ReadOnly: true, Handler: s.suggestMetrics},
		{Name: "dry_run_step", Description: "Reservado para executar um step isolado em modo controlado.", ReadOnly: false, Handler: s.writeToolDisabled},
		{Name: "reprocess_execution", Description: "Reservado para reprocessar a partir do cursor salvo.", ReadOnly: false, Handler: s.writeToolDisabled},
	} {
		tool.InputSchema = map[string]any{"type": "object", "additionalProperties": true}
		s.tools[tool.Name] = tool
	}
}

func (s *mcpServer) run(ctx context.Context) error {
	if !s.cfg.MCP.Enabled {
		return nil
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, req *http.Request) {
		setMCPCORSHeaders(w)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "mode": s.cfg.MCP.Mode})
	})
	mux.HandleFunc("/mcp", s.handleMCP)

	server := &http.Server{Addr: s.cfg.MCP.Bind, Handler: mux}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	s.logger.Info("mcp gateway listening", slog.String("bind", s.cfg.MCP.Bind), slog.String("mode", s.cfg.MCP.Mode))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *mcpServer) handleMCP(w http.ResponseWriter, req *http.Request) {
	setMCPCORSHeaders(w)
	if req.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(mcpError(nil, -32600, "method not allowed"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if !s.authorized(req) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(mcpError(nil, -32001, "unauthorized"))
		return
	}

	var request mcpRequest
	if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(mcpError(nil, -32700, err.Error()))
		return
	}

	switch request.Method {
	case "tools/list":
		_ = json.NewEncoder(w).Encode(mcpResult(request.ID, map[string]any{"tools": s.toolList()}))
	case "tools/call":
		result, err := s.callTool(req.Context(), request.Params)
		if err != nil {
			_ = json.NewEncoder(w).Encode(mcpError(request.ID, -32000, err.Error()))
			return
		}
		_ = json.NewEncoder(w).Encode(mcpResult(request.ID, result))
	default:
		_ = json.NewEncoder(w).Encode(mcpError(request.ID, -32601, "method not found"))
	}
}

func setMCPCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, traceparent, X-Trace-ID, X-Correlation-ID")
	w.Header().Set("Access-Control-Expose-Headers", "traceparent, X-Trace-ID")
}

func (s *mcpServer) authorized(req *http.Request) bool {
	if s.cfg.MCP.Auth.Type != "api_key" {
		return true
	}
	expected := strings.TrimSpace(os.Getenv(s.cfg.MCP.Auth.Env))
	if expected == "" {
		return false
	}
	token := strings.TrimSpace(req.Header.Get("Authorization"))
	token = strings.TrimPrefix(token, "Bearer ")
	if token == "" {
		token = strings.TrimSpace(req.Header.Get("X-API-Key"))
	}
	return token == expected
}

func (s *mcpServer) toolList() []map[string]any {
	names := make([]string, 0, len(s.tools))
	for name := range s.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		tool := s.tools[name]
		out = append(out, map[string]any{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
			"annotations": map[string]any{"readOnlyHint": tool.ReadOnly},
		})
	}
	return out
}

func (s *mcpServer) callTool(ctx context.Context, raw json.RawMessage) (map[string]any, error) {
	var params mcpCallParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, err
	}
	tool, ok := s.tools[params.Name]
	if !ok {
		return nil, fmt.Errorf("unknown tool %q", params.Name)
	}
	if !tool.ReadOnly && s.cfg.MCP.Mode == "readonly" {
		return nil, fmt.Errorf("tool %q requires maintenance mode", params.Name)
	}
	result, err := tool.Handler(ctx, params.Arguments)
	if err != nil {
		return nil, err
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return map[string]any{
		"content":           []map[string]string{{"type": "text", "text": string(data)}},
		"structuredContent": result,
	}, nil
}

func (s *mcpServer) listHandlers(ctx context.Context, args map[string]any) (any, error) {
	return registeredHandlerSpecs(), nil
}

func (s *mcpServer) validateWorkflow(ctx context.Context, args map[string]any) (any, error) {
	workflow, source, err := s.workflowFromArgs(args)
	if err != nil {
		return nil, err
	}
	issues := validateWorkflowDiagnostics(workflow)
	return map[string]any{"valid": len(source) > 0 && !hasDiagnosticError(issues), "issues": issues}, nil
}

func (s *mcpServer) explainWorkflow(ctx context.Context, args map[string]any) (any, error) {
	workflow, _, err := s.workflowFromArgs(args)
	if err != nil {
		return nil, err
	}
	steps := make([]map[string]any, 0, len(workflow.Steps))
	for index, step := range workflow.Steps {
		steps = append(steps, map[string]any{
			"index":       index,
			"id":          step.ID,
			"handler":     step.Name,
			"integration": step.Name == "graphql_enrich" || step.Name == "rest_call" || step.Name == "notify",
			"control":     step.Name == "condition" || step.Name == "assert" || step.Name == "jump_if" || step.Name == "cel",
			"target":      firstString(step.Params, "target", "to"),
		})
	}
	return map[string]any{
		"name":                workflow.Name,
		"description":         workflow.Description,
		"version":             workflow.Version,
		"error_policy":        workflow.ErrorPolicy,
		"message_id_path":     workflow.MessageIDPath,
		"correlation_id_path": workflow.CorrelationIDPath,
		"steps":               steps,
	}, nil
}

func (s *mcpServer) exportWorkflow(ctx context.Context, args map[string]any) (any, error) {
	workflow, _, err := s.workflowFromArgs(args)
	if err != nil {
		return nil, err
	}
	data, err := yaml.Marshal(workflow)
	if err != nil {
		return nil, err
	}
	return map[string]any{"workflow": workflow.Name, "yaml": string(data)}, nil
}

func (s *mcpServer) getExecution(ctx context.Context, args map[string]any) (any, error) {
	filter := snapshotFilterFromArgs(args)
	if filter.MessageID != "" {
		snapshot, err := s.store.Load(ctx, filter.MessageID)
		if err != nil {
			return nil, err
		}
		return snapshot, nil
	}
	snapshots, err := s.listSnapshots(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(snapshots) == 0 {
		return nil, slip.ErrStateNotFound
	}
	return snapshots[0], nil
}

func (s *mcpServer) listStateSnapshots(ctx context.Context, args map[string]any) (any, error) {
	return s.listSnapshots(ctx, snapshotFilterFromArgs(args))
}

func (s *mcpServer) listSnapshots(ctx context.Context, filter slip.SnapshotFilter) ([]slip.MessageSnapshot, error) {
	lister, ok := s.store.(slip.StateSnapshotLister)
	if !ok {
		return nil, fmt.Errorf("state store does not support listing")
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	return lister.List(ctx, filter)
}

func (s *mcpServer) writeToolDisabled(ctx context.Context, args map[string]any) (any, error) {
	return nil, fmt.Errorf("tool reserved for a later controlled implementation")
}

func (s *mcpServer) workflowFromArgs(args map[string]any) (*WorkflowConfig, string, error) {
	if path := stringArg(args, "path"); path != "" {
		workflow, err := loadWorkflowConfig(path)
		return workflow, path, err
	}
	if source := stringArg(args, "yaml"); source != "" {
		var workflow WorkflowConfig
		if err := yaml.Unmarshal([]byte(source), &workflow); err != nil {
			return nil, "", err
		}
		applyWorkflowDefaults(&workflow)
		if err := validateWorkflowConfig(&workflow); err != nil {
			return nil, "", err
		}
		return &workflow, source, nil
	}
	return s.workflow, "configured", nil
}

func snapshotFilterFromArgs(args map[string]any) slip.SnapshotFilter {
	filter := slip.SnapshotFilter{
		MessageID:     stringArg(args, "message_id"),
		CorrelationID: stringArg(args, "correlation_id"),
		TraceID:       stringArg(args, "trace_id"),
		Workflow:      stringArg(args, "workflow"),
		Status:        stringArg(args, "status"),
		Limit:         intArg(args, "limit"),
	}
	if value := stringArg(args, "from"); value != "" {
		filter.From, _ = time.Parse(time.RFC3339, value)
	}
	if value := stringArg(args, "to"); value != "" {
		filter.To, _ = time.Parse(time.RFC3339, value)
	}
	return filter
}

func validateWorkflowDiagnostics(workflow *WorkflowConfig) []map[string]string {
	issues := make([]map[string]string, 0)
	handlers := handlerNameSet()
	if workflow == nil {
		return []map[string]string{{"level": "error", "message": "workflow ausente"}}
	}
	if strings.TrimSpace(workflow.Name) == "" {
		issues = append(issue(issues, "error", "campo name e obrigatorio"))
	}
	if len(workflow.Steps) == 0 {
		issues = append(issue(issues, "error", "workflow sem steps"))
	}
	targets := map[string]bool{}
	for _, step := range workflow.Steps {
		if step.ID != "" {
			targets[step.ID] = true
		}
		targets[step.Name] = true
	}
	for index, step := range workflow.Steps {
		label := fmt.Sprintf("steps[%d]", index)
		if !handlers[step.Name] {
			issues = append(issue(issues, "error", label+".name nao registrado: "+step.Name))
		}
		if step.Name == "jump_if" {
			to := firstString(step.Params, "to")
			if to == "" || !targets[to] {
				issues = append(issue(issues, "error", label+".params.to nao aponta para step conhecido"))
			}
		}
		if step.Resilience.OnFailure.Action == "jump" {
			to := step.Resilience.OnFailure.To
			if to == "" || !targets[to] {
				issues = append(issue(issues, "error", label+".resilience.on_failure.to nao aponta para step conhecido"))
			}
		}
	}
	return issues
}

func registeredHandlerSpecs() []map[string]any {
	specs := []map[string]any{
		{"name": "validate", "params": []string{"required", "stop_on_failure"}},
		{"name": "condition", "params": []string{"field", "equals", "not_equals"}},
		{"name": "assert", "params": []string{"all", "any", "field", "exists", "message"}},
		{"name": "compute", "params": []string{"target", "value"}},
		{"name": "cel", "params": []string{"expr", "on_false", "to"}},
		{"name": "filter_array", "params": []string{"source", "target", "where", "expr"}},
		{"name": "array_transform", "params": []string{"source", "target", "filters", "updates", "nested"}},
		{"name": "enrich", "params": []string{"data", "prefix"}},
		{"name": "transform", "params": []string{"field", "operation", "target"}},
		{"name": "notify", "params": []string{"channel", "recipient", "template"}},
		{"name": "log", "params": []string{"level", "message", "fields", "data"}},
		{"name": "audit", "params": []string{"event", "fields"}},
		{"name": "datadog_metric", "params": []string{"metric", "value", "type", "tags", "api_key", "api_url"}},
		{"name": "graphql_enrich", "params": []string{"endpoint", "query", "variables", "target", "result_path", "required"}},
		{"name": "jump_if", "params": []string{"field", "exists", "equals", "not_equals", "min_items", "max_items", "to"}},
		{"name": "rest_call", "params": []string{"base_url", "endpoint", "method", "target", "required"}},
		{"name": "aws_action", "params": []string{"service", "action", "region", "endpoint", "target", "required"}},
		{"name": "workflow_ref", "params": []string{"file", "path", "workflow", "prefix"}},
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i]["name"].(string) < specs[j]["name"].(string) })
	return specs
}

func handlerNameSet() map[string]bool {
	out := map[string]bool{}
	for _, spec := range registeredHandlerSpecs() {
		out[spec["name"].(string)] = true
	}
	return out
}

func issue(issues []map[string]string, level, message string) []map[string]string {
	return append(issues, map[string]string{"level": level, "message": message})
}

func hasDiagnosticError(issues []map[string]string) bool {
	for _, item := range issues {
		if item["level"] == "error" {
			return true
		}
	}
	return false
}

func stringArg(args map[string]any, key string) string {
	value, _ := args[key].(string)
	return strings.TrimSpace(value)
}

func intArg(args map[string]any, key string) int {
	switch value := args[key].(type) {
	case int:
		return value
	case float64:
		return int(value)
	case json.Number:
		n, _ := value.Int64()
		return int(n)
	default:
		return 0
	}
}

func firstString(params map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := params[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func mcpResult(id any, result any) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "result": result}
}

func mcpError(id any, code int, message string) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": code, "message": message}}
}
