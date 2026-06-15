package framework

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raywall/routing-slip-pattern/app/source"
)

func TestRuntimeLoadsSourcesAndProcessesIdempotently(t *testing.T) {
	runtime, err := New(context.Background(), Options{
		ConfigSource: source.Inline(`service: {name: test}
state_store: {type: memory}
idempotency: {enabled: true}`),
		WorkflowSource: source.Inline(`name: test
message_id_path: event_id
correlation_id_path: correlation_id
steps:
  - id: validate
    name: validate
    params: {required: [event_id]}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{"event_id": "evt-1"}
	first, err := runtime.Process(context.Background(), payload)
	if err != nil {
		t.Fatal(err)
	}
	second, err := runtime.Process(context.Background(), payload)
	if err != nil {
		t.Fatal(err)
	}
	if first.CorrelationID == "" || second.CorrelationID != first.CorrelationID {
		t.Fatalf("correlation mismatch: %q %q", first.CorrelationID, second.CorrelationID)
	}
}

func TestRuntimeMCPExposesBusinessRules(t *testing.T) {
	runtime, err := New(context.Background(), Options{
		ConfigSource: source.Inline(`service: {name: test}`),
		WorkflowSource: source.Inline(`name: test
business_rules:
  - id: product-required
    name: Product required
    status: active
steps:
  - name: enrich
    params: {data: {ok: true}}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_business_rules","arguments":{}}}`))
	rec := httptest.NewRecorder()
	runtime.MCPHandler().ServeHTTP(rec, req)
	if !bytes.Contains(rec.Body.Bytes(), []byte("product-required")) {
		t.Fatalf("unexpected MCP response: %s", rec.Body.String())
	}
}
