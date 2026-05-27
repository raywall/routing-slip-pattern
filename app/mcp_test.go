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

func testMCPServer() *mcpServer {
	cfg := &AppConfig{}
	applyConfigDefaults(cfg)
	cfg.MCP.Enabled = true
	workflow := &WorkflowConfig{Name: "test", Steps: []StepConfig{{Name: "validate", Params: map[string]any{"required": []any{"id"}}}}}
	return newMCPServer(cfg, workflow, slip.NewMemoryStateStore(), slog.Default())
}
