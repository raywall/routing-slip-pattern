package handlers

import (
	"context"
	"testing"

	"github.com/raywall/routing-slip-pattern/slip"
)

func TestNotificationHandlerPropagatesTraceHeaders(t *testing.T) {
	msg := slip.NewMessage("msg-1", map[string]any{"name": "Ava"})
	msg.CorrelationID = "corr-1"
	msg.TraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	msg.SpanID = "00f067aa0ba902b7"
	msg.ParentSpanID = "9f3a4c7b12e64010"
	msg.Headers["traceparent"] = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

	var got map[string]string
	handler := &NotificationHandler{
		SendWithHeaders: func(channel, recipient, body string, headers map[string]string) error {
			got = headers
			return nil
		},
	}

	proceed, err := handler.Handle(context.Background(), msg, map[string]any{
		"channel":   "webhook",
		"recipient": "https://example.test/hook",
		"template":  "hello {name}",
	})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if !proceed {
		t.Fatal("expected handler to proceed")
	}
	if got["traceparent"] != msg.Headers["traceparent"] {
		t.Fatalf("traceparent = %q", got["traceparent"])
	}
	if got["X-Trace-ID"] != msg.TraceID {
		t.Fatalf("X-Trace-ID = %q", got["X-Trace-ID"])
	}
	if got["X-Correlation-ID"] != msg.CorrelationID {
		t.Fatalf("X-Correlation-ID = %q", got["X-Correlation-ID"])
	}
}
