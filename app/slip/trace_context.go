package slip

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	traceparentHeader  = "traceparent"
	traceIDHeader      = "trace_id"
	spanIDHeader       = "span_id"
	parentSpanIDHeader = "parent_span_id"
)

type traceContextKey struct{}

// TraceContext carries the W3C trace identifiers used by workflow handlers.
type TraceContext struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
	TraceFlags   string
}

// TraceContextFromContext returns the trace context currently attached to ctx.
func TraceContextFromContext(ctx context.Context) (TraceContext, bool) {
	value, ok := ctx.Value(traceContextKey{}).(TraceContext)
	return value, ok && value.TraceID != "" && value.SpanID != ""
}

// ContextWithTraceContext attaches trace identifiers to ctx.
func ContextWithTraceContext(ctx context.Context, trace TraceContext) context.Context {
	return context.WithValue(ctx, traceContextKey{}, trace)
}

// EnsureTraceContext extracts an incoming traceparent when present or creates a
// new trace. The message is updated so metrics, snapshots and responses can
// expose the same identifiers.
func EnsureTraceContext(ctx context.Context, msg *Message) (context.Context, TraceContext) {
	if trace, ok := TraceContextFromContext(ctx); ok {
		applyTraceContext(msg, trace)
		return ctx, trace
	}
	if trace, ok := traceContextFromMessage(msg); ok {
		applyTraceContext(msg, trace)
		return ContextWithTraceContext(ctx, trace), trace
	}
	trace := TraceContext{TraceID: newHexID(16), SpanID: newHexID(8), TraceFlags: "01"}
	applyTraceContext(msg, trace)
	return ContextWithTraceContext(ctx, trace), trace
}

// StartTraceSpan creates a child span context for the current step. It keeps the
// same trace_id and rotates span_id, matching the W3C traceparent shape.
func StartTraceSpan(ctx context.Context, msg *Message) (context.Context, TraceContext) {
	ctx, parent := EnsureTraceContext(ctx, msg)
	trace := TraceContext{
		TraceID:      parent.TraceID,
		ParentSpanID: parent.SpanID,
		SpanID:       newHexID(8),
		TraceFlags:   parent.TraceFlags,
	}
	applyTraceContext(msg, trace)
	return ContextWithTraceContext(ctx, trace), trace
}

// Traceparent serializes a trace context using the W3C traceparent header.
func Traceparent(trace TraceContext) string {
	flags := trace.TraceFlags
	if flags == "" {
		flags = "01"
	}
	return fmt.Sprintf("00-%s-%s-%s", trace.TraceID, trace.SpanID, flags)
}

func traceContextFromMessage(msg *Message) (TraceContext, bool) {
	if msg == nil {
		return TraceContext{}, false
	}
	if raw := msg.Headers[traceparentHeader]; raw != "" {
		if trace, ok := ParseTraceparent(raw); ok {
			return trace, true
		}
	}
	if raw := msg.Headers["Traceparent"]; raw != "" {
		if trace, ok := ParseTraceparent(raw); ok {
			return trace, true
		}
	}
	traceID := firstNonEmpty(msg.TraceID, msg.Headers[traceIDHeader], msg.Headers["X-Trace-ID"], msg.Headers["x-trace-id"])
	spanID := firstNonEmpty(msg.SpanID, msg.Headers[spanIDHeader])
	if traceID == "" {
		return TraceContext{}, false
	}
	if spanID == "" {
		spanID = newHexID(8)
	}
	return TraceContext{
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: firstNonEmpty(msg.ParentSpanID, msg.Headers[parentSpanIDHeader]),
		TraceFlags:   "01",
	}, true
}

// ParseTraceparent reads a W3C traceparent header.
func ParseTraceparent(value string) (TraceContext, bool) {
	parts := strings.Split(strings.TrimSpace(value), "-")
	if len(parts) != 4 || parts[0] != "00" || len(parts[1]) != 32 || len(parts[2]) != 16 {
		return TraceContext{}, false
	}
	return TraceContext{TraceID: parts[1], SpanID: parts[2], TraceFlags: parts[3]}, true
}

func applyTraceContext(msg *Message, trace TraceContext) {
	if msg == nil {
		return
	}
	msg.mu.Lock()
	defer msg.mu.Unlock()
	msg.TraceID = trace.TraceID
	msg.SpanID = trace.SpanID
	msg.ParentSpanID = trace.ParentSpanID
	if msg.Headers == nil {
		msg.Headers = map[string]string{}
	}
	msg.Headers[traceparentHeader] = Traceparent(trace)
	msg.Headers[traceIDHeader] = trace.TraceID
	msg.Headers[spanIDHeader] = trace.SpanID
	if trace.ParentSpanID != "" {
		msg.Headers[parentSpanIDHeader] = trace.ParentSpanID
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func newHexID(bytesLen int) string {
	data := make([]byte, bytesLen)
	if _, err := rand.Read(data); err != nil {
		return strings.Repeat("0", bytesLen*2)
	}
	return hex.EncodeToString(data)
}
