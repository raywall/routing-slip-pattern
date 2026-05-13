package slip

import (
	"context"
	"fmt"
	"log/slog"
	"time"
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
	m.emit(ctx, msg, "routing_slip.step.started", "running", 1, "event", 0)
	proceed, err := m.next.Handle(ctx, msg, params)
	duration := time.Since(start)

	status := "success"
	name := "routing_slip.step.completed"
	if err != nil {
		status = "failed"
		name = "routing_slip.step.failed"
	} else if !proceed {
		status = "stopped"
		name = "routing_slip.step.stopped"
	}
	m.emit(ctx, msg, name, status, 1, "event", duration)
	m.emit(ctx, msg, "routing_slip.step.duration_ms", status, float64(duration.Milliseconds()), "ms", duration)
	return proceed, err
}

func (m *metricsHandler) emit(ctx context.Context, msg *Message, name, status string, value float64, unit string, duration time.Duration) {
	if m.emitter == nil {
		return
	}
	tags := map[string]string{
		"message_id": msg.ID,
		"handler":    m.next.Name(),
	}
	for k, v := range m.opts.Tags {
		tags[k] = v
	}
	if correlationID := msg.GetString("correlation_id"); correlationID != "" {
		tags["correlation_id"] = correlationID
	}
	if customerID := msg.GetString("customer_id"); customerID != "" {
		tags["customer_id"] = customerID
	}
	if productID := msg.GetString("product_id"); productID != "" {
		tags["product_id"] = productID
	}
	if duration > 0 {
		tags["duration_ms"] = fmt.Sprintf("%d", duration.Milliseconds())
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
