package slip

import (
	"context"
	"testing"
)

func TestTracingMiddlewarePopulatesMessageAndHistory(t *testing.T) {
	router := NewRouter(WithMiddleware(TracingMiddleware()))
	router.MustRegister(testSetHandler{name: "set", key: "processed", value: true})

	msg := NewMessage("msg-1", map[string]any{"correlation_id": "corr-1"})
	msg.AttachSlip([]StepDef{{Name: "set"}})

	if err := router.Process(context.Background(), msg); err != nil {
		t.Fatalf("Process returned error: %v", err)
	}
	if msg.TraceID == "" {
		t.Fatal("expected message trace id")
	}
	if msg.SpanID == "" {
		t.Fatal("expected message span id")
	}
	if msg.Headers["traceparent"] == "" {
		t.Fatal("expected traceparent header")
	}
	if len(msg.History) != 1 {
		t.Fatalf("expected one history entry, got %d", len(msg.History))
	}
	entry := msg.History[0]
	if entry.TraceID != msg.TraceID {
		t.Fatalf("history trace id = %q, want %q", entry.TraceID, msg.TraceID)
	}
	if entry.SpanID != msg.SpanID {
		t.Fatalf("history span id = %q, want %q", entry.SpanID, msg.SpanID)
	}
	if entry.Status != "success" {
		t.Fatalf("history status = %q, want success", entry.Status)
	}
}

func TestMessageSnapshotPreservesTraceFields(t *testing.T) {
	msg := NewMessage("msg-1", nil)
	msg.CorrelationID = "corr-1"
	msg.TraceID = "trace-1"
	msg.SpanID = "span-1"
	msg.ParentSpanID = "parent-1"
	msg.Attempt = 3

	restored := MessageFromSnapshot(msg.Snapshot())

	if restored.CorrelationID != msg.CorrelationID ||
		restored.TraceID != msg.TraceID ||
		restored.SpanID != msg.SpanID ||
		restored.ParentSpanID != msg.ParentSpanID ||
		restored.Attempt != msg.Attempt {
		t.Fatalf("restored trace fields do not match snapshot")
	}
}
