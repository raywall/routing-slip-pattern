package slip

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// LoggingMiddleware logs the duration of each handler call.
func LoggingMiddleware(logger *slog.Logger) Middleware {
	return func(next Handler) Handler {
		return &loggingHandler{next: next, logger: logger}
	}
}

type loggingHandler struct {
	next   Handler
	logger *slog.Logger
}

func (l *loggingHandler) Name() string { return l.next.Name() }

func (l *loggingHandler) Handle(ctx context.Context, msg *Message, params map[string]any) (bool, error) {
	start := time.Now()
	proceed, err := l.next.Handle(ctx, msg, params)
	l.logger.Debug("middleware: handler timing",
		slog.String("handler", l.next.Name()),
		slog.Duration("elapsed", time.Since(start)),
		slog.Bool("proceed", proceed),
	)
	return proceed, err
}

// RecoveryMiddleware catches panics inside handlers and converts them to errors.
func RecoveryMiddleware() Middleware {
	return func(next Handler) Handler {
		return &recoveryHandler{next: next}
	}
}

type recoveryHandler struct {
	next Handler
}

func (r *recoveryHandler) Name() string { return r.next.Name() }

func (r *recoveryHandler) Handle(ctx context.Context, msg *Message, params map[string]any) (proceed bool, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("panic in handler %q: %v", r.next.Name(), rec)
			proceed = false
		}
	}()
	return r.next.Handle(ctx, msg, params)
}

// TracingMiddleware creates a W3C trace span context for each handler call.
// It intentionally keeps the implementation lightweight: the trace identifiers
// are propagated through context/message headers and can be exported to OTel by
// infrastructure adapters in later phases.
func TracingMiddleware() Middleware {
	return func(next Handler) Handler {
		return &tracingHandler{next: next}
	}
}

type tracingHandler struct {
	next Handler
}

func (t *tracingHandler) Name() string { return t.next.Name() }

func (t *tracingHandler) Handle(ctx context.Context, msg *Message, params map[string]any) (bool, error) {
	ctx, trace := StartTraceSpan(ctx, msg)
	if msg.Attempt <= 0 {
		msg.Attempt = 1
	}
	msg.Headers["traceparent"] = Traceparent(trace)
	tracer := otel.Tracer("routing-slip-pattern/slip")
	ctx, span := tracer.Start(ctx, "routing-slip.step."+t.next.Name())
	span.SetAttributes(
		attribute.String("routing_slip.message_id", msg.ID),
		attribute.String("routing_slip.correlation_id", msg.CorrelationID),
		attribute.String("routing_slip.trace_id", msg.TraceID),
		attribute.String("routing_slip.span_id", msg.SpanID),
		attribute.String("routing_slip.handler", t.next.Name()),
		attribute.Int("routing_slip.attempt", msg.Attempt),
	)
	defer span.End()

	proceed, err := t.next.Handle(ctx, msg, params)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	return proceed, err
}

// MetricsMiddleware emits business metrics for each routing slip step.
func MetricsMiddleware(emitter MetricsEmitter, opts MetricsOptions, logger *slog.Logger) Middleware {
	return func(next Handler) Handler {
		return &metricsHandler{next: next, emitter: emitter, opts: opts, logger: logger}
	}
}

type metricsHandler struct {
	next    Handler
	emitter MetricsEmitter
	opts    MetricsOptions
	logger  *slog.Logger
}

func (m *metricsHandler) Name() string { return m.next.Name() }

func (m *metricsHandler) Handle(ctx context.Context, msg *Message, params map[string]any) (bool, error) {
	start := time.Now()
	input := summarizeInput(m.next.Name(), msg, params)
	rule := summarizeRule(m.next.Name(), params)
	m.emit(ctx, msg, "routing_slip.step.started", "running", 1, "event", 0, input, rule, "", "")
	proceed, err := m.next.Handle(ctx, msg, params)
	duration := time.Since(start)
	output := summarizeOutput(m.next.Name(), msg, params)

	status := "success"
	name := "routing_slip.step.completed"
	failureReason := ""
	if err != nil {
		status = "failed"
		name = "routing_slip.step.failed"
		failureReason = err.Error()
	} else if !proceed {
		status = "stopped"
		name = "routing_slip.step.stopped"
		failureReason = "handler requested workflow stop"
	}
	m.emit(ctx, msg, name, status, 1, "event", duration, input, rule, output, failureReason)
	m.emit(ctx, msg, "routing_slip.step.duration_ms", status, float64(duration.Milliseconds()), "ms", duration, input, rule, output, failureReason)
	return proceed, err
}

func (m *metricsHandler) emit(ctx context.Context, msg *Message, name, status string, value float64, unit string, duration time.Duration, input, rule, output, failureReason string) {
	if m.emitter == nil {
		return
	}
	tags := map[string]string{
		"message_id":  msg.ID,
		"handler":     m.next.Name(),
		"step_index":  fmt.Sprintf("%d", max(0, msg.currentCursor()-1)),
		"total_steps": fmt.Sprintf("%d", len(msg.slip)),
	}
	for k, v := range m.opts.Tags {
		tags[k] = v
	}
	if correlationID := msg.GetString("correlation_id"); correlationID != "" {
		tags["correlation_id"] = correlationID
	}
	if msg.CorrelationID != "" {
		tags["correlation_id"] = msg.CorrelationID
	}
	if msg.TraceID != "" {
		tags["trace_id"] = msg.TraceID
	}
	if msg.SpanID != "" {
		tags["span_id"] = msg.SpanID
	}
	if msg.ParentSpanID != "" {
		tags["parent_span_id"] = msg.ParentSpanID
	}
	if msg.Attempt > 0 {
		tags["attempt"] = fmt.Sprintf("%d", msg.Attempt)
	}
	if customerID := msg.GetString("customer_id"); customerID != "" {
		tags["customer_id"] = customerID
	}
	if productID := msg.GetString("product_id"); productID != "" {
		tags["product_id"] = productID
	}
	addTagFromPath(tags, msg, "pagamento_id", "payload.pagamento_id")
	addTagFromPath(tags, msg, "pedido_id", "payload.pedido_id", "pedido.pedido_id")
	addTagFromPath(tags, msg, "id_cliente", "pedido.cliente_id", "customer_id")
	addTagFromPath(tags, msg, "cliente_id", "pedido.cliente_id", "customer_id")
	addTagFromPath(tags, msg, "nota_fiscal_id", "nota_fiscal.nota_fiscal_id")
	addTagFromPath(tags, msg, "expedicao_id", "expedicao.expedicao_id")
	addTagFromPath(tags, msg, "codigo_rastreio", "expedicao.codigo_rastreio")
	if duration > 0 {
		tags["duration_ms"] = fmt.Sprintf("%d", duration.Milliseconds())
	}
	if input != "" {
		tags["input_value"] = input
	}
	if rule != "" {
		tags["rule_applied"] = rule
	}
	if output != "" {
		tags["output_value"] = output
	}
	if failureReason != "" {
		tags["failure_reason"] = failureReason
	}

	event := MetricEvent{
		Name:      name,
		Kind:      "count",
		Value:     value,
		Unit:      unit,
		Segment:   m.opts.Segment,
		Workflow:  m.opts.Workflow,
		Step:      m.next.Name(),
		Status:    status,
		Source:    m.opts.Source,
		TraceID:   msg.TraceID,
		SpanID:    msg.SpanID,
		Tags:      tags,
		Timestamp: time.Now(),
	}
	if name == "routing_slip.step.duration_ms" {
		event.Kind = "gauge"
	}
	if err := m.emitter.Emit(ctx, event); err != nil && m.logger != nil {
		m.logger.Warn("metrics: emit failed", slog.String("error", err.Error()))
	}
}

func summarizeInput(step string, msg *Message, params map[string]any) string {
	switch step {
	case "validate":
		return summarizeRequiredValues(msg, params)
	case "condition":
		field := fmt.Sprintf("%v", params["field"])
		value, _ := msg.GetPath(field)
		return compactJSON(map[string]any{"field": field, "value": value})
	case "graphql_enrich":
		return compactJSON(map[string]any{"query": params["query"], "variables": params["variables"]})
	case "rest_call":
		return compactJSON(map[string]any{"method": params["method"], "endpoint": params["endpoint"], "body": params["body"]})
	case "transform":
		field := fmt.Sprintf("%v", params["field"])
		value, _ := msg.GetPath(field)
		return compactJSON(map[string]any{"field": field, "value": value})
	case "enrich":
		return compactJSON(params["data"])
	default:
		return compactJSON(params)
	}
}

func summarizeRule(step string, params map[string]any) string {
	switch step {
	case "validate":
		return compactJSON(map[string]any{"required": params["required"], "stop_on_failure": params["stop_on_failure"]})
	case "condition":
		return compactJSON(map[string]any{"field": params["field"], "equals": params["equals"]})
	case "graphql_enrich":
		return compactJSON(map[string]any{"target": params["target"], "result_path": params["result_path"], "required": params["required"]})
	case "rest_call":
		return compactJSON(map[string]any{"method": params["method"], "endpoint": params["endpoint"], "target": params["target"], "required": params["required"]})
	case "transform":
		return compactJSON(map[string]any{"field": params["field"], "operation": params["operation"], "target": params["target"]})
	case "enrich":
		return compactJSON(map[string]any{"prefix": params["prefix"]})
	default:
		return compactJSON(params)
	}
}

func summarizeOutput(step string, msg *Message, params map[string]any) string {
	switch step {
	case "validate":
		return compactJSON(map[string]any{"validation_passed": msg.Payload["validation_passed"]})
	case "condition":
		return compactJSON(map[string]any{"gate_stopped": msg.Payload["gate_stopped"]})
	case "graphql_enrich", "rest_call":
		target := fmt.Sprintf("%v", params["target"])
		value, _ := msg.GetPath(target)
		return compactJSON(map[string]any{target: value})
	case "transform":
		target := fmt.Sprintf("%v", params["target"])
		if target == "" || target == "<nil>" {
			target = fmt.Sprintf("%v", params["field"])
		}
		value, _ := msg.GetPath(target)
		return compactJSON(map[string]any{target: value})
	case "enrich":
		return compactJSON(map[string]any{"payload": msg.Payload})
	default:
		return compactJSON(map[string]any{"status": "processed"})
	}
}

func summarizeRequiredValues(msg *Message, params map[string]any) string {
	required, ok := params["required"].([]string)
	if !ok {
		return compactJSON(params["required"])
	}
	values := make(map[string]any, len(required))
	for _, field := range required {
		value, _ := msg.GetPath(field)
		values[field] = value
	}
	return compactJSON(values)
}

func compactJSON(value any) string {
	if value == nil {
		return ""
	}
	data, err := json.Marshal(RedactSensitive(value))
	if err != nil {
		return truncateText(fmt.Sprintf("%v", value), 900)
	}
	return truncateText(string(data), 900)
}

func truncateText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit-3] + "..."
}

func addTagFromPath(tags map[string]string, msg *Message, tag string, paths ...string) {
	for _, path := range paths {
		if value, ok := msg.GetPath(path); ok && value != nil {
			text := fmt.Sprintf("%v", value)
			if text != "" && text != "<nil>" {
				tags[tag] = text
				return
			}
		}
	}
}
