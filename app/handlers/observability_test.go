package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raywall/routing-slip-pattern/slip"
)

func TestLogHandlerRecordsLastLog(t *testing.T) {
	msg := slip.NewMessage("msg-1", map[string]any{"order_id": "ORD-1"})
	msg.CorrelationID = "corr-1"

	ok, err := LogHandler{}.Handle(context.Background(), msg, map[string]any{
		"level":   "info",
		"message": "Order {order_id} accepted",
		"fields":  []any{"order_id"},
	})
	if err != nil || !ok {
		t.Fatalf("expected log handler success, ok=%v err=%v", ok, err)
	}
	if got, _ := msg.Get("last_log_level"); got != "info" {
		t.Fatalf("last_log_level = %v", got)
	}
}

func TestDatadogMetricHandlerPostsSeries(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("DD-API-KEY"); got != "test-key" {
			t.Fatalf("DD-API-KEY = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	msg := slip.NewMessage("msg-1", map[string]any{"status": "success"})
	msg.CorrelationID = "corr-1"
	ok, err := DatadogMetricHandler{}.Handle(context.Background(), msg, map[string]any{
		"api_url": server.URL,
		"api_key": "test-key",
		"metric":  "workflow.completed",
		"value":   2,
		"type":    "count",
		"tags": map[string]any{
			"status": "{status}",
		},
	})
	if err != nil || !ok {
		t.Fatalf("expected metric handler success, ok=%v err=%v", ok, err)
	}
	series := payload["series"].([]any)
	first := series[0].(map[string]any)
	if first["metric"] != "workflow.completed" {
		t.Fatalf("metric = %v", first["metric"])
	}
	tags := first["tags"].([]any)
	if len(tags) < 2 {
		t.Fatalf("expected correlation and custom tags, got %v", tags)
	}
}
