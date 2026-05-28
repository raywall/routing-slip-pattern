package main

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type plannerEndpoint struct {
	Name        string
	Method      string
	URL         string
	Description string
}

func (s *mcpServer) planWorkflow(ctx context.Context, args map[string]any) (any, error) {
	description := stringArg(args, "description")
	if description == "" {
		description = "Workflow gerado pelo planner MCP"
	}
	eventName := stringArg(args, "event_name")
	if eventName == "" {
		eventName = detectEventName(description)
	}
	workflowName := cleanStepID(stringArg(args, "name"))
	if workflowName == "" {
		workflowName = cleanStepID(eventName)
	}
	if workflowName == "" {
		workflowName = "planned-workflow"
	}

	required := requiredFieldsFromArgs(args)
	if len(required) == 0 {
		required = []string{"correlation_id", "event_name"}
	}
	endpoints := plannerEndpoints(args["endpoints"])
	steps := []StepConfig{
		{
			ID:   "validate-input",
			Name: "validate",
			Params: map[string]any{
				"required": required,
			},
		},
		{
			ID:   "audit-received",
			Name: "audit",
			Params: map[string]any{
				"event":  workflowName + ".received",
				"fields": []string{"correlation_id", "event_name"},
			},
		},
	}

	for index, endpoint := range endpoints {
		step := endpointToStep(endpoint, index)
		steps = append(steps, step)
	}

	steps = append(steps, StepConfig{
		ID:   "audit-completed",
		Name: "audit",
		Params: map[string]any{
			"event":  workflowName + ".completed",
			"fields": []string{"correlation_id", "event_name"},
		},
	})

	workflow := WorkflowConfig{
		Name:              workflowName,
		Description:       description,
		Version:           "draft",
		ErrorPolicy:       "stop",
		MessageIDPath:     firstNonEmptyString(required, "correlation_id"),
		CorrelationIDPath: "correlation_id",
		Steps:             steps,
	}
	data, err := yaml.Marshal(workflow)
	if err != nil {
		return nil, err
	}
	risks := idempotencyRisks(workflow)
	metrics := metricsForWorkflow(workflow)
	return map[string]any{
		"workflow":        workflowAsMap(workflow),
		"yaml":            string(data),
		"test_payload":    payloadForWorkflow(workflow, eventName),
		"bruno_requests":  brunoRequestsForWorkflow(workflow.Name),
		"idempotency":     risks,
		"metrics":         metrics,
		"audit_points":    auditPoints(workflow),
		"decision_notes":  plannerDecisionNotes(workflow, endpoints),
		"requires_review": true,
	}, nil
}

func workflowAsMap(workflow WorkflowConfig) map[string]any {
	data, err := yaml.Marshal(workflow)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := yaml.Unmarshal(data, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func (s *mcpServer) suggestHandlers(ctx context.Context, args map[string]any) (any, error) {
	text := strings.ToLower(stringArg(args, "description") + " " + strings.Join(stringSliceArg(args, "capabilities"), " "))
	suggestions := make([]map[string]any, 0)
	add := func(handler, reason string) {
		suggestions = append(suggestions, map[string]any{"handler": handler, "reason": reason})
	}
	if hasAny(text, "valid", "obrig", "required", "schema") {
		add("validate", "garante campos obrigatorios antes de efeitos externos")
	}
	if hasAny(text, "api", "http", "rest", "endpoint", "webhook") {
		add("rest_call", "integra com APIs REST sem criar handler customizado")
	}
	if hasAny(text, "graphql", "enriquec", "consulta", "compose") {
		add("graphql_enrich", "centraliza enriquecimento por GraphQL connector")
	}
	if hasAny(text, "decis", "condic", "if", "regra") {
		add("condition", "interrompe fluxo quando uma regra simples nao e atendida")
		add("assert", "trava o processamento quando criterios obrigatorios falham")
	}
	if hasAny(text, "calcular", "derivar", "flag", "compute") {
		add("compute", "cria atributos derivados para decisoes posteriores")
	}
	if hasAny(text, "pular", "rotear", "branch", "jump") {
		add("jump_if", "redireciona o cursor para outra etapa")
	}
	if hasAny(text, "lista", "array", "filtrar", "itens") {
		add("filter_array", "remove itens de arrays conforme criterio")
	}
	if hasAny(text, "auditoria", "audit", "observ", "metric") {
		add("audit", "cria ponto explicavel de auditoria e metricas")
	}
	if len(suggestions) == 0 {
		add("validate", "todo workflow deve iniciar com validacao minima")
		add("audit", "todo workflow deve ter pontos de auditoria")
	}
	return map[string]any{"suggestions": suggestions}, nil
}

func (s *mcpServer) generateTestPayload(ctx context.Context, args map[string]any) (any, error) {
	workflow, _, err := s.workflowFromArgs(args)
	if err != nil {
		return nil, err
	}
	eventName := stringArg(args, "event_name")
	if eventName == "" {
		eventName = workflow.Name + ".requested"
	}
	return payloadForWorkflow(*workflow, eventName), nil
}

func (s *mcpServer) generateBrunoCollection(ctx context.Context, args map[string]any) (any, error) {
	name := stringArg(args, "workflow")
	if name == "" && s.workflow != nil {
		name = s.workflow.Name
	}
	if name == "" {
		name = "planned-workflow"
	}
	return brunoRequestsForWorkflow(name), nil
}

func (s *mcpServer) assessIdempotency(ctx context.Context, args map[string]any) (any, error) {
	workflow, _, err := s.workflowFromArgs(args)
	if err != nil {
		return nil, err
	}
	return map[string]any{"risks": idempotencyRisks(*workflow)}, nil
}

func (s *mcpServer) suggestMetrics(ctx context.Context, args map[string]any) (any, error) {
	workflow, _, err := s.workflowFromArgs(args)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"metrics":      metricsForWorkflow(*workflow),
		"audit_points": auditPoints(*workflow),
	}, nil
}

func endpointToStep(endpoint plannerEndpoint, index int) StepConfig {
	name := cleanStepID(endpoint.Name)
	if name == "" {
		name = fmt.Sprintf("integration-%02d", index+1)
	}
	if strings.Contains(strings.ToLower(endpoint.Description+" "+endpoint.URL), "graphql") {
		return StepConfig{
			ID:   name,
			Name: "graphql_enrich",
			Params: map[string]any{
				"target":      name,
				"query":       "query { dataSources { status } }",
				"result_path": "dataSources",
				"required":    true,
			},
		}
	}
	method := strings.ToUpper(endpoint.Method)
	if method == "" {
		method = "GET"
	}
	baseURL, path := splitEndpointURL(endpoint.URL)
	return StepConfig{
		ID:   name,
		Name: "rest_call",
		Params: map[string]any{
			"base_url": baseURL,
			"endpoint": path,
			"method":   method,
			"target":   name,
			"required": true,
		},
	}
}

func plannerEndpoints(raw any) []plannerEndpoint {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]plannerEndpoint, 0, len(items))
	for index, item := range items {
		values, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := stringMapValue(values, "name")
		if name == "" {
			name = fmt.Sprintf("integration-%02d", index+1)
		}
		out = append(out, plannerEndpoint{
			Name:        name,
			Method:      stringMapValue(values, "method"),
			URL:         stringMapValue(values, "url"),
			Description: stringMapValue(values, "description"),
		})
	}
	return out
}

func requiredFieldsFromArgs(args map[string]any) []string {
	fields := stringSliceArg(args, "required_fields")
	if len(fields) > 0 {
		return fields
	}
	if event, ok := args["event"].(map[string]any); ok {
		out := make([]string, 0, len(event))
		for key := range event {
			out = append(out, key)
		}
		sort.Strings(out)
		return out
	}
	return nil
}

func payloadForWorkflow(workflow WorkflowConfig, eventName string) map[string]any {
	payload := map[string]any{
		"correlation_id": newCorrelationUUID(),
		"event_name":     eventName,
		"received_at":    "2026-05-27T12:00:00Z",
	}
	for _, step := range workflow.Steps {
		if step.Name != "validate" {
			continue
		}
		for _, field := range anySlice(step.Params["required"]) {
			key, ok := field.(string)
			if !ok || payload[key] != nil {
				continue
			}
			payload[key] = sampleValueForField(key)
		}
	}
	return payload
}

func idempotencyRisks(workflow WorkflowConfig) []map[string]any {
	risks := make([]map[string]any, 0)
	for index, step := range workflow.Steps {
		if step.Name == "notify" || step.Name == "rest_call" {
			risks = append(risks, map[string]any{
				"step":           step.ID,
				"index":          index,
				"handler":        step.Name,
				"risk":           "possivel efeito externo repetido em reprocessamento",
				"recommendation": "usar state_store.idempotency.enabled e id estavel no step",
			})
		}
		if step.Name == "graphql_enrich" {
			risks = append(risks, map[string]any{
				"step":           step.ID,
				"index":          index,
				"handler":        step.Name,
				"risk":           "dados externos podem mudar entre execucoes",
				"recommendation": "persistir snapshot e avaliar se o enriquecimento deve ser reexecutado",
			})
		}
	}
	return risks
}

func metricsForWorkflow(workflow WorkflowConfig) []map[string]any {
	return []map[string]any{
		{"name": "workflow_started_total", "tags": []string{"workflow", "correlation_id"}},
		{"name": "workflow_completed_total", "tags": []string{"workflow", "status"}},
		{"name": "workflow_failed_total", "tags": []string{"workflow", "step", "handler"}},
		{"name": "workflow_step_duration_ms", "tags": []string{"workflow", "step", "handler"}},
		{"name": "workflow_reprocess_total", "tags": []string{"workflow", "message_id"}},
		{"name": "workflow_idempotent_skip_total", "tags": []string{"workflow", "step"}},
	}
}

func auditPoints(workflow WorkflowConfig) []map[string]any {
	points := []map[string]any{{"event": workflow.Name + ".received", "after": "validate-input"}}
	for _, step := range workflow.Steps {
		if step.Name == "graphql_enrich" || step.Name == "rest_call" || step.Name == "notify" {
			points = append(points, map[string]any{"event": workflow.Name + "." + step.ID + ".completed", "after": step.ID})
		}
	}
	points = append(points, map[string]any{"event": workflow.Name + ".completed", "after": "last-step"})
	return points
}

func brunoRequestsForWorkflow(workflow string) []map[string]any {
	return []map[string]any{
		{"name": "Executar workflow", "method": "POST", "url": "http://localhost:8088/process", "body": map[string]any{"correlation_id": newCorrelationUUID(), "event_name": workflow + ".requested"}},
		{"name": "Listar tools MCP", "method": "POST", "url": "http://localhost:9091/mcp", "body": map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}},
		{"name": "Explicar workflow via MCP", "method": "POST", "url": "http://localhost:9091/mcp", "body": map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{"name": "explain_workflow", "arguments": map[string]any{}}}},
	}
}

func plannerDecisionNotes(workflow WorkflowConfig, endpoints []plannerEndpoint) []string {
	notes := []string{
		"workflow gerado como rascunho; revise nomes, paths e regras antes de executar",
		"nenhum arquivo foi alterado pelo planner",
		"steps com efeito externo devem ter id estavel para idempotencia",
	}
	if len(endpoints) > 0 {
		notes = append(notes, "endpoints informados foram convertidos em rest_call ou graphql_enrich")
	}
	if workflow.MessageIDPath == "correlation_id" {
		notes = append(notes, "message_id_path usa correlation_id por falta de identificador mais especifico")
	}
	return notes
}

func splitEndpointURL(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "https://api.example.test", "/resource"
	}
	re := regexp.MustCompile(`^(https?://[^/]+)(/.*)?$`)
	matches := re.FindStringSubmatch(value)
	if len(matches) == 0 {
		return "https://api.example.test", value
	}
	path := matches[2]
	if path == "" {
		path = "/"
	}
	return matches[1], path
}

func detectEventName(description string) string {
	text := strings.ToLower(description)
	switch {
	case strings.Contains(text, "pedido"):
		return "order.requested"
	case strings.Contains(text, "pagamento"):
		return "payment.received"
	case strings.Contains(text, "estoque"):
		return "inventory.changed"
	default:
		return "workflow.requested"
	}
}

func sampleValueForField(field string) any {
	lower := strings.ToLower(field)
	switch {
	case strings.Contains(lower, "id"):
		return "ID-1001"
	case strings.Contains(lower, "valor"), strings.Contains(lower, "amount"), strings.Contains(lower, "total"):
		return 100.5
	case strings.Contains(lower, "quantidade"), strings.Contains(lower, "quantity"):
		return 1
	case strings.Contains(lower, "data"), strings.Contains(lower, "date"):
		return "2026-05-27"
	default:
		return "sample"
	}
}

func hasAny(text string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func stringSliceArg(args map[string]any, key string) []string {
	items := anySlice(args[key])
	out := make([]string, 0, len(items))
	for _, item := range items {
		if value, ok := item.(string); ok && strings.TrimSpace(value) != "" {
			out = append(out, strings.TrimSpace(value))
		}
	}
	return out
}

func anySlice(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case []string:
		out := make([]any, len(typed))
		for index, item := range typed {
			out[index] = item
		}
		return out
	default:
		return nil
	}
}

func stringMapValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func firstNonEmptyString(values []string, fallback string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return fallback
}
