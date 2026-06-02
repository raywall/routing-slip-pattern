package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raywall/routing-slip-pattern/slip"
)

func TestMCPToolsList(t *testing.T) {
	server := testMCPServer()
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	result := body["result"].(map[string]any)
	tools := result["tools"].([]any)
	if len(tools) == 0 {
		t.Fatal("expected tools")
	}
}

func TestMCPCORSPreflight(t *testing.T) {
	server := testMCPServer()
	req := httptest.NewRequest(http.MethodOptions, "/mcp", nil)
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatal("expected CORS headers")
	}
}

func TestMCPValidateWorkflow(t *testing.T) {
	server := testMCPServer()
	payload := `{
		"jsonrpc":"2.0",
		"id":2,
		"method":"tools/call",
		"params":{
			"name":"validate_workflow",
			"arguments":{
				"yaml":"name: test\nsteps:\n  - name: validate\n    params:\n      required:\n        - id\n"
			}
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct {
		Result struct {
			StructuredContent struct {
				Valid bool `json:"valid"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !body.Result.StructuredContent.Valid {
		t.Fatalf("expected valid workflow response: %s", rec.Body.String())
	}
}

func TestMCPWriteToolRequiresMaintenanceMode(t *testing.T) {
	server := testMCPServer()
	_, err := server.callTool(context.Background(), []byte(`{"name":"reprocess_execution","arguments":{}}`))
	if err == nil {
		t.Fatal("expected readonly rejection")
	}
}

func TestMCPPlanWorkflow(t *testing.T) {
	server := testMCPServer()
	result, err := server.callTool(context.Background(), []byte(`{
		"name": "plan_workflow",
		"arguments": {
			"name": "Catalog Sync",
			"description": "Recebe evento de catalogo, consulta API REST de produto e audita o resultado",
			"required_fields": ["correlation_id", "product_id"],
			"endpoints": [
				{"name": "product-api", "method": "GET", "url": "https://api.example.test/products/{product_id}"}
			]
		}
	}`))
	if err != nil {
		t.Fatalf("plan workflow: %v", err)
	}
	content := result["structuredContent"].(map[string]any)
	if content["yaml"] == "" {
		t.Fatal("expected yaml draft")
	}
	if content["requires_review"] != true {
		t.Fatal("expected requires_review true")
	}
}

func TestMCPGenerateWorkflowFromBusinessRules(t *testing.T) {
	server := testMCPServer()
	result, err := server.callTool(context.Background(), []byte(`{
		"name": "generate_workflow_from_business_rules",
		"arguments": {
			"workflow_name": "Order Review",
			"business_rules": [
				{
					"rule_id": "order_total_positive",
					"status": "ACTIVE",
					"execution_order": 1,
					"human_context": {
						"name": "Total positivo",
						"description": "O campo {order.total} deve existir antes da aprovacao."
					},
					"technical_metadata": {
						"dependencies": [
							{"type": "business_rule", "Rule_id": "catalog_available", "Relation": "depends_on"},
							{"type": "system", "Name": "sqs", "Component": "order-events", "Action": "consume"}
						],
						"observability": {
							"datadog_monitor_ids": [123, 456],
							"custom_metrics": {"name": "routing_slip.order.total_checked", "type": "gauge", "tags": ["env:test"]},
							"log_markers": ["total-check"]
						}
					}
				}
			]
		}
	}`))
	if err != nil {
		t.Fatalf("generate from rules: %v", err)
	}
	content := result["structuredContent"].(map[string]any)
	if content["yaml"] == "" {
		t.Fatal("expected generated yaml")
	}
	payload := content["test_payload"].(map[string]any)
	if payload["correlation_id"] == "" {
		t.Fatal("expected payload correlation_id")
	}
	rules := content["active_rules"].([]map[string]any)
	if len(rules) != 1 || len(rules[0]["datadog_monitor_ids"].([]string)) != 2 {
		t.Fatalf("expected datadog monitor ids in active rule, got %#v", rules)
	}
}

func TestMCPValidateWorkflowAgainstBusinessRules(t *testing.T) {
	server := testMCPServer()
	result, err := server.callTool(context.Background(), []byte(`{
		"name": "validate_workflow_against_business_rules",
		"arguments": {
			"yaml": "name: test\nsteps:\n  - name: validate\n    params:\n      required:\n        - correlation_id\n",
			"business_rules": [
				{
					"rule_id": "must_audit_order",
					"status": "ACTIVE",
					"human_context": {"name": "Auditar pedido", "description": "Auditar o pedido."}
				}
			]
		}
	}`))
	if err != nil {
		t.Fatalf("validate against rules: %v", err)
	}
	content := result["structuredContent"].(map[string]any)
	if content["valid"] != true {
		t.Fatal("expected valid response when only rule coverage warnings exist")
	}
	issues := content["issues"].([]map[string]string)
	if len(issues) == 0 {
		t.Fatal("expected rule coverage issue")
	}
	if issues[0]["level"] != "warn" {
		t.Fatalf("expected warning, got %v", issues[0])
	}
}

func testMCPServer() *mcpServer {
	cfg := &AppConfig{}
	applyConfigDefaults(cfg)
	cfg.MCP.Enabled = true
	workflow := &WorkflowConfig{Name: "test", Steps: []StepConfig{{Name: "validate", Params: map[string]any{"required": []any{"id"}}}}}
	return newMCPServer(cfg, workflow, slip.NewMemoryStateStore(), slog.Default())
}
