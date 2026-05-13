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
