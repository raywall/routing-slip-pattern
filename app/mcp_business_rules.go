package main

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type mcpBusinessRule struct {
	ID              string
	Name            string
	Status          string
	Description     string
	AILogic         string
	ExecutionOrder  int
	RequiredFields  []string
	CustomMetrics   []string
	LogMarkers      []string
	Dependencies    []string
	SourceReference map[string]any
}

func (s *mcpServer) generateWorkflowFromBusinessRules(ctx context.Context, args map[string]any) (any, error) {
	rules := businessRulesFromArgs(args)
	if len(rules) == 0 {
		return nil, fmt.Errorf("business_rules or rules_text is required")
	}
	activeRules := activeBusinessRules(rules)
	if len(activeRules) == 0 {
		activeRules = rules
	}

	workflowName := cleanStepID(firstNonEmptyString([]string{stringArg(args, "workflow_name"), stringArg(args, "name")}, "business-rules-workflow"))
	description := stringArg(args, "description")
	if description == "" {
		description = "Workflow gerado a partir de regras de negocio."
	}
	required := mergeRequiredFields(activeRules, stringSliceArg(args, "required_fields"))
	if len(required) == 0 {
		required = []string{"correlation_id", "event_name"}
	}

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

	coverage := make([]map[string]any, 0, len(activeRules))
	for _, rule := range activeRules {
		ruleID := cleanStepID(rule.ID)
		if ruleID == "" {
			ruleID = fmt.Sprintf("rule-%02d", len(coverage)+1)
		}
		steps = append(steps,
			StepConfig{
				ID:   ruleID + "-log",
				Name: "log",
				Params: map[string]any{
					"level":   "info",
					"message": "Avaliando regra " + rule.ID + " - " + rule.Name,
					"fields":  []string{"correlation_id", "event_name"},
					"data": map[string]any{
						"rule_id":     rule.ID,
						"rule_status": rule.Status,
					},
					"required": false,
				},
			},
			StepConfig{
				ID:   ruleID + "-assert",
				Name: "cel",
				Params: map[string]any{
					"expr":     businessRuleExpression(rule),
					"target":   "business_rules." + ruleID + ".passed",
					"on_false": "error",
					"message":  "Regra de negocio nao atendida: " + rule.ID,
				},
			},
			StepConfig{
				ID:   ruleID + "-audit",
				Name: "audit",
				Params: map[string]any{
					"event":  workflowName + "." + ruleID + ".checked",
					"fields": append([]string{"correlation_id", "event_name"}, rule.RequiredFields...),
				},
			},
		)
		for _, marker := range rule.LogMarkers {
			steps = append(steps, StepConfig{
				ID:   cleanStepID(ruleID + "-" + marker + "-log"),
				Name: "log",
				Params: map[string]any{
					"level":    "info",
					"message":  marker,
					"fields":   []string{"correlation_id", "event_name"},
					"data":     map[string]any{"rule_id": rule.ID},
					"required": false,
				},
			})
		}
		for _, metric := range rule.CustomMetrics {
			steps = append(steps, StepConfig{
				ID:   cleanStepID(ruleID + "-" + metric + "-metric"),
				Name: "datadog_metric",
				Params: map[string]any{
					"metric":   metric,
					"type":     "count",
					"value":    1,
					"required": false,
					"tags": map[string]any{
						"rule_id":  rule.ID,
						"workflow": workflowName,
					},
				},
			})
		}
		coverage = append(coverage, map[string]any{
			"rule_id":         rule.ID,
			"covered_by":      []string{ruleID + "-log", ruleID + "-assert", ruleID + "-audit"},
			"requires_review": true,
			"note":            "A expressao CEL foi gerada como ponto inicial e deve ser refinada conforme a regra real.",
		})
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
	payload := payloadForWorkflow(workflow, firstNonEmptyString([]string{stringArg(args, "event_name"), workflowName + ".requested"}, "workflow.requested"))
	return map[string]any{
		"workflow":        workflowAsMap(workflow),
		"yaml":            string(data),
		"test_payload":    payload,
		"coverage":        coverage,
		"active_rules":    businessRulesAsMaps(activeRules),
		"requires_review": true,
		"decision_notes": []string{
			"o rascunho foi gerado a partir das regras ativas",
			"expressoes CEL foram criadas como ponto inicial para refinamento",
			"logs, auditorias e metricas foram adicionados quando havia metadados de observabilidade",
		},
	}, nil
}

func (s *mcpServer) validateWorkflowAgainstBusinessRules(ctx context.Context, args map[string]any) (any, error) {
	workflow, _, err := s.workflowFromArgs(args)
	if err != nil {
		return nil, err
	}
	rules := activeBusinessRules(businessRulesFromArgs(args))
	issues := validateBusinessRuleCoverage(workflow, rules)
	return map[string]any{
		"valid":         !hasDiagnosticError(issues),
		"issues":        issues,
		"rules_checked": len(rules),
		"coverage":      businessRuleCoverage(workflow, rules),
	}, nil
}

func businessRulesFromArgs(args map[string]any) []mcpBusinessRule {
	rules := make([]mcpBusinessRule, 0)
	for _, item := range anySlice(args["business_rules"]) {
		values, ok := item.(map[string]any)
		if !ok {
			continue
		}
		rule := businessRuleFromMap(values)
		if rule.ID != "" || rule.Name != "" || rule.Description != "" {
			rules = append(rules, rule)
		}
	}
	if len(rules) > 0 {
		sortBusinessRules(rules)
		return rules
	}
	text := strings.TrimSpace(stringArg(args, "rules_text"))
	if text == "" {
		text = strings.TrimSpace(stringArg(args, "description"))
	}
	if text == "" {
		return nil
	}
	chunks := splitRuleText(text)
	for index, chunk := range chunks {
		id := fmt.Sprintf("rule-%02d", index+1)
		rules = append(rules, mcpBusinessRule{
			ID:             id,
			Name:           firstLine(chunk),
			Status:         "ACTIVE",
			Description:    chunk,
			ExecutionOrder: index + 1,
			RequiredFields: inferRequiredFields(chunk),
		})
	}
	sortBusinessRules(rules)
	return rules
}

func businessRuleFromMap(values map[string]any) mcpBusinessRule {
	human := mapArg(values["human_context"])
	engineering := mapArg(values["engineering_context"])
	metadata := mapArg(values["technical_metadata"])
	observability := mapArg(metadata["observability"])
	rule := mcpBusinessRule{
		ID:              firstNonEmptyString([]string{stringMapValue(values, "rule_id"), stringMapValue(values, "id")}, ""),
		Name:            firstNonEmptyString([]string{stringMapValue(human, "name"), stringMapValue(values, "name")}, ""),
		Status:          strings.ToUpper(firstNonEmptyString([]string{stringMapValue(values, "status")}, "ACTIVE")),
		Description:     firstNonEmptyString([]string{stringMapValue(human, "description"), stringMapValue(values, "description")}, ""),
		AILogic:         firstNonEmptyString([]string{stringMapValue(values, "ai_logic"), stringMapValue(values, "logic")}, ""),
		ExecutionOrder:  intArgFromMap(values, "execution_order"),
		RequiredFields:  mergeStringSlices(stringSliceFromAny(values["required_inputs"]), stringSliceFromAny(values["required_fields"]), stringSliceFromAny(engineering["required_fields"])),
		CustomMetrics:   mergeStringSlices(stringSliceFromAny(observability["custom_metric"]), stringSliceFromAny(observability["custom_metrics"])),
		LogMarkers:      mergeStringSlices(stringSliceFromAny(observability["log_markers"]), stringSliceFromAny(observability["logs"])),
		Dependencies:    dependencyIDs(metadata["dependencies"]),
		SourceReference: values,
	}
	if len(rule.RequiredFields) == 0 {
		rule.RequiredFields = inferRequiredFields(rule.Description + "\n" + rule.AILogic)
	}
	if rule.Name == "" {
		rule.Name = rule.ID
	}
	if rule.ID == "" {
		rule.ID = cleanStepID(rule.Name)
	}
	return rule
}

func validateBusinessRuleCoverage(workflow *WorkflowConfig, rules []mcpBusinessRule) []map[string]string {
	issues := []map[string]string{}
	if len(rules) == 0 {
		return issues
	}
	text := workflowSearchText(workflow)
	for _, rule := range rules {
		tokens := businessRuleTokens(rule)
		if !containsAnyToken(text, tokens) {
			issues = append(issue(issues, "error", fmt.Sprintf("regra ativa %q nao esta referenciada no workflow", rule.ID)))
		}
		for _, field := range rule.RequiredFields {
			if !workflowRequiresField(workflow, field) && !strings.Contains(text, strings.ToLower(field)) {
				issues = append(issue(issues, "warn", fmt.Sprintf("regra %q espera o campo %q, mas ele nao aparece no workflow", rule.ID, field)))
			}
		}
		for _, metric := range rule.CustomMetrics {
			if !workflowHasDatadogMetric(workflow, metric) {
				issues = append(issue(issues, "warn", fmt.Sprintf("regra %q declara metrica %q, mas nao ha datadog_metric correspondente", rule.ID, metric)))
			}
		}
		for _, marker := range rule.LogMarkers {
			if !workflowHasLogMarker(workflow, marker) {
				issues = append(issue(issues, "warn", fmt.Sprintf("regra %q declara log marker %q, mas nao ha log correspondente", rule.ID, marker)))
			}
		}
		for _, dependency := range rule.Dependencies {
			if dependency != "" && !ruleExists(rules, dependency) {
				issues = append(issue(issues, "warn", fmt.Sprintf("regra %q depende de %q, que nao esta ativa no projeto", rule.ID, dependency)))
			}
		}
	}
	return issues
}

func businessRuleCoverage(workflow *WorkflowConfig, rules []mcpBusinessRule) []map[string]any {
	text := workflowSearchText(workflow)
	out := make([]map[string]any, 0, len(rules))
	for _, rule := range rules {
		out = append(out, map[string]any{
			"rule_id":          rule.ID,
			"covered":          containsAnyToken(text, businessRuleTokens(rule)),
			"required_fields":  rule.RequiredFields,
			"custom_metrics":   rule.CustomMetrics,
			"log_markers":      rule.LogMarkers,
			"covered_steps":    coveredStepsForRule(workflow, rule),
			"dependencies":     rule.Dependencies,
			"execution_order":  rule.ExecutionOrder,
			"business_summary": firstNonEmptyString([]string{rule.Name, firstLine(rule.Description)}, rule.ID),
		})
	}
	return out
}

func activeBusinessRules(rules []mcpBusinessRule) []mcpBusinessRule {
	out := make([]mcpBusinessRule, 0, len(rules))
	for _, rule := range rules {
		status := strings.ToUpper(strings.TrimSpace(rule.Status))
		if status == "" || status == "ACTIVE" {
			out = append(out, rule)
		}
	}
	sortBusinessRules(out)
	return out
}

func businessRuleExpression(rule mcpBusinessRule) string {
	fields := rule.RequiredFields
	if len(fields) == 0 {
		return "true"
	}
	expressions := make([]string, 0, len(fields))
	for _, field := range fields {
		clean := strings.TrimSpace(field)
		if clean == "" {
			continue
		}
		expressions = append(expressions, clean+" != null")
	}
	if len(expressions) == 0 {
		return "true"
	}
	return strings.Join(expressions, " && ")
}

func mergeRequiredFields(rules []mcpBusinessRule, base []string) []string {
	fields := mergeStringSlices(base, []string{"correlation_id", "event_name"})
	for _, rule := range rules {
		fields = mergeStringSlices(fields, rule.RequiredFields)
	}
	sort.Strings(fields)
	return fields
}

func businessRulesAsMaps(rules []mcpBusinessRule) []map[string]any {
	out := make([]map[string]any, 0, len(rules))
	for _, rule := range rules {
		out = append(out, map[string]any{
			"rule_id":         rule.ID,
			"name":            rule.Name,
			"status":          rule.Status,
			"execution_order": rule.ExecutionOrder,
			"required_fields": rule.RequiredFields,
			"custom_metrics":  rule.CustomMetrics,
			"log_markers":     rule.LogMarkers,
			"dependencies":    rule.Dependencies,
		})
	}
	return out
}

func workflowSearchText(workflow *WorkflowConfig) string {
	data, _ := yaml.Marshal(workflow)
	return strings.ToLower(string(data))
}

func workflowRequiresField(workflow *WorkflowConfig, field string) bool {
	for _, step := range workflow.Steps {
		if step.Name != "validate" {
			continue
		}
		for _, item := range anySlice(step.Params["required"]) {
			if fmt.Sprintf("%v", item) == field {
				return true
			}
		}
	}
	return false
}

func workflowHasDatadogMetric(workflow *WorkflowConfig, metric string) bool {
	metric = strings.ToLower(strings.TrimSpace(metric))
	for _, step := range workflow.Steps {
		if step.Name == "datadog_metric" && strings.ToLower(fmt.Sprintf("%v", step.Params["metric"])) == metric {
			return true
		}
	}
	return false
}

func workflowHasLogMarker(workflow *WorkflowConfig, marker string) bool {
	marker = strings.ToLower(strings.TrimSpace(marker))
	for _, step := range workflow.Steps {
		if step.Name == "log" && strings.Contains(strings.ToLower(fmt.Sprintf("%v", step.Params["message"])), marker) {
			return true
		}
	}
	return false
}

func coveredStepsForRule(workflow *WorkflowConfig, rule mcpBusinessRule) []string {
	tokens := businessRuleTokens(rule)
	out := []string{}
	for _, step := range workflow.Steps {
		data, _ := yaml.Marshal(step)
		if containsAnyToken(strings.ToLower(string(data)), tokens) {
			out = append(out, firstNonEmptyString([]string{step.ID, step.Name}, "step"))
		}
	}
	return out
}

func businessRuleTokens(rule mcpBusinessRule) []string {
	return []string{strings.ToLower(rule.ID), strings.ToLower(cleanStepID(rule.ID)), strings.ToLower(rule.Name)}
}

func containsAnyToken(text string, tokens []string) bool {
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token != "" && strings.Contains(text, token) {
			return true
		}
	}
	return false
}

func ruleExists(rules []mcpBusinessRule, id string) bool {
	for _, rule := range rules {
		if rule.ID == id {
			return true
		}
	}
	return false
}

func splitRuleText(text string) []string {
	lines := strings.Split(text, "\n")
	chunks := []string{}
	current := []string{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if len(current) > 0 {
				chunks = append(chunks, strings.Join(current, "\n"))
				current = nil
			}
			continue
		}
		current = append(current, trimmed)
	}
	if len(current) > 0 {
		chunks = append(chunks, strings.Join(current, "\n"))
	}
	if len(chunks) == 0 && strings.TrimSpace(text) != "" {
		chunks = append(chunks, strings.TrimSpace(text))
	}
	return chunks
}

func inferRequiredFields(text string) []string {
	re := regexp.MustCompile(`\{([a-zA-Z0-9_.-]+)\}`)
	matches := re.FindAllStringSubmatch(text, -1)
	fields := []string{}
	for _, match := range matches {
		if len(match) > 1 {
			fields = append(fields, match[1])
		}
	}
	return mergeStringSlices(fields)
}

func firstLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) != "" {
			return strings.TrimSpace(line)
		}
	}
	return "Regra de negocio"
}

func sortBusinessRules(rules []mcpBusinessRule) {
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].ExecutionOrder == rules[j].ExecutionOrder {
			return rules[i].ID < rules[j].ID
		}
		return rules[i].ExecutionOrder < rules[j].ExecutionOrder
	})
}

func mapArg(value any) map[string]any {
	if values, ok := value.(map[string]any); ok {
		return values
	}
	return map[string]any{}
}

func intArgFromMap(values map[string]any, key string) int {
	switch value := values[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func stringSliceFromAny(value any) []string {
	out := []string{}
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if text := strings.TrimSpace(fmt.Sprintf("%v", item)); text != "" {
				out = append(out, text)
			}
		}
	case []string:
		for _, item := range typed {
			if text := strings.TrimSpace(item); text != "" {
				out = append(out, text)
			}
		}
	case string:
		for _, item := range strings.Split(typed, ",") {
			if text := strings.TrimSpace(item); text != "" {
				out = append(out, text)
			}
		}
	}
	return out
}

func mergeStringSlices(lists ...[]string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, list := range lists {
		for _, item := range list {
			value := strings.TrimSpace(item)
			if value == "" || seen[value] {
				continue
			}
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func dependencyIDs(value any) []string {
	out := []string{}
	for _, item := range anySlice(value) {
		switch typed := item.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				out = append(out, strings.TrimSpace(typed))
			}
		case map[string]any:
			id := firstNonEmptyString([]string{stringMapValue(typed, "rule_id"), stringMapValue(typed, "system")}, "")
			if id != "" {
				out = append(out, id)
			}
		}
	}
	return out
}
