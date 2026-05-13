// Package slip implements the Routing Slip enterprise integration pattern.
// A Routing Slip attaches a list of processing steps to a message at runtime,
// allowing dynamic, configurable workflows without hard-coded pipelines.
package slip

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Message
// ---------------------------------------------------------------------------

// Message is the unit of work that travels through the workflow.
// It carries a payload, metadata, the routing slip itself, and an audit trail.
type Message struct {
	mu sync.RWMutex

	ID        string
	CreatedAt time.Time
	Payload   map[string]any // mutable data transformed by handlers
	Headers   map[string]string
	slip      []StepDef // ordered list of steps to execute
	cursor    int       // next step index
	History   []HistoryEntry
	Errors    []StepError
}

// NewMessage creates a new Message with the given ID and initial payload.
func NewMessage(id string, payload map[string]any) *Message {
	if payload == nil {
		payload = make(map[string]any)
	}
	return &Message{
		ID:        id,
		CreatedAt: time.Now(),
		Payload:   payload,
		Headers:   make(map[string]string),
	}
}

// AttachSlip appends the step definitions to the message's routing slip.
// Steps are executed in the order they are attached.
func (m *Message) AttachSlip(steps []StepDef) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.slip = append(m.slip, steps...)
}

// NextStep returns the next step definition and advances the cursor.
// Returns false when the slip is exhausted.
func (m *Message) NextStep() (StepDef, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cursor >= len(m.slip) {
		return StepDef{}, false
	}
	s := m.slip[m.cursor]
	m.cursor++
	return s, true
}

// Set writes a value into the payload (thread-safe).
func (m *Message) Set(key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Payload[key] = value
}

// Get reads a value from the payload (thread-safe).
func (m *Message) Get(key string) (any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.Payload[key]
	return v, ok
}

// GetString reads a string value from the payload.
func (m *Message) GetString(key string) string {
	v, ok := m.Get(key)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// AddHistory records the result of a completed step.
func (m *Message) AddHistory(entry HistoryEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.History = append(m.History, entry)
}

// AddError records an error that occurred during a step.
func (m *Message) AddError(e StepError) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Errors = append(m.Errors, e)
}

// HasErrors reports whether any step produced an error.
func (m *Message) HasErrors() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.Errors) > 0
}

// RemainingSteps returns the number of steps not yet executed.
func (m *Message) RemainingSteps() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.slip) - m.cursor
}

// ---------------------------------------------------------------------------
// Routing Slip definition
// ---------------------------------------------------------------------------

// StepDef describes a single step in the routing slip.
type StepDef struct {
	Name   string         // human-readable step name
	Params map[string]any // arbitrary parameters passed to the handler
}

// HistoryEntry is an audit record for a completed step.
type HistoryEntry struct {
	Step      string
	StartedAt time.Time
	Duration  time.Duration
	Skipped   bool
}

// StepError captures a failure in a specific step.
type StepError struct {
	Step      string
	Err       error
	Timestamp time.Time
}

// ---------------------------------------------------------------------------
// Handler interface
// ---------------------------------------------------------------------------

// Handler is the interface every processing step must implement.
type Handler interface {
	// Name returns the unique identifier used to register this handler.
	Name() string

	// Handle processes the message. It may mutate msg.Payload and/or
	// msg.Headers. Return (false, nil) to skip remaining steps gracefully.
	// Return (_, err) to record an error; behaviour depends on ErrorPolicy.
	Handle(ctx context.Context, msg *Message, params map[string]any) (proceed bool, err error)
}

// ---------------------------------------------------------------------------
// Router
// ---------------------------------------------------------------------------

// ErrorPolicy controls what happens when a handler returns an error.
type ErrorPolicy int

const (
	// StopOnError halts the workflow on the first error (default).
	StopOnError ErrorPolicy = iota
	// ContinueOnError records the error but keeps processing remaining steps.
	ContinueOnError
	// SkipOnError records the error and skips to the next step silently.
	SkipOnError
)

// RouterOption is a functional option for Router.
type RouterOption func(*Router)

// WithLogger sets a custom slog.Logger.
func WithLogger(l *slog.Logger) RouterOption {
	return func(r *Router) { r.logger = l }
}

// WithErrorPolicy sets the error-handling policy.
func WithErrorPolicy(p ErrorPolicy) RouterOption {
	return func(r *Router) { r.errorPolicy = p }
}

// WithMiddleware adds middleware that wraps every handler call.
func WithMiddleware(mw ...Middleware) RouterOption {
	return func(r *Router) { r.middleware = append(r.middleware, mw...) }
}

// Middleware wraps a Handler, enabling cross-cutting concerns (logging, metrics, etc.).
type Middleware func(next Handler) Handler

// Router executes a Message through its attached routing slip.
type Router struct {
	handlers    map[string]Handler
	errorPolicy ErrorPolicy
	logger      *slog.Logger
	middleware  []Middleware
}

// NewRouter creates a Router with the given options.
func NewRouter(opts ...RouterOption) *Router {
	r := &Router{
		handlers:    make(map[string]Handler),
		errorPolicy: StopOnError,
		logger:      slog.Default(),
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Register adds a handler to the router's registry.
// Panics if a handler with the same name is already registered.
func (r *Router) Register(h Handler) {
	name := h.Name()
	if _, exists := r.handlers[name]; exists {
		panic(fmt.Sprintf("routing-slip: handler %q already registered", name))
	}
	r.handlers[name] = r.applyMiddleware(h)
}

// MustRegister is an alias for Register (panics on duplicate).
func (r *Router) MustRegister(h Handler) { r.Register(h) }

// Process executes the routing slip attached to msg.
// It returns an error only when errorPolicy == StopOnError and a step fails.
func (r *Router) Process(ctx context.Context, msg *Message) error {
	r.logger.Info("router: starting workflow",
		slog.String("message_id", msg.ID),
		slog.Int("total_steps", len(msg.slip)),
	)

	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("context cancelled before completing slip: %w", err)
		}

		step, ok := msg.NextStep()
		if !ok {
			break
		}

		h, found := r.handlers[step.Name]
		if !found {
			err := fmt.Errorf("no handler registered for step %q", step.Name)
			msg.AddError(StepError{Step: step.Name, Err: err, Timestamp: time.Now()})
			if r.errorPolicy == StopOnError {
				return err
			}
			continue
		}

		start := time.Now()
		r.logger.Info("router: executing step",
			slog.String("step", step.Name),
			slog.String("message_id", msg.ID),
		)

		proceed, err := h.Handle(ctx, msg, step.Params)
		duration := time.Since(start)

		if err != nil {
			stepErr := StepError{Step: step.Name, Err: err, Timestamp: time.Now()}
			msg.AddError(stepErr)
			r.logger.Error("router: step error",
				slog.String("step", step.Name),
				slog.String("error", err.Error()),
			)
			switch r.errorPolicy {
			case StopOnError:
				return fmt.Errorf("step %q: %w", step.Name, err)
			case ContinueOnError:
				msg.AddHistory(HistoryEntry{Step: step.Name, StartedAt: start, Duration: duration})
				continue
			case SkipOnError:
				msg.AddHistory(HistoryEntry{Step: step.Name, StartedAt: start, Duration: duration, Skipped: true})
				continue
			}
		}

		msg.AddHistory(HistoryEntry{Step: step.Name, StartedAt: start, Duration: duration})

		if !proceed {
			r.logger.Info("router: step requested stop", slog.String("step", step.Name))
			break
		}
	}

	r.logger.Info("router: workflow complete",
		slog.String("message_id", msg.ID),
		slog.Int("steps_executed", len(msg.History)),
		slog.Bool("has_errors", msg.HasErrors()),
	)
	return nil
}

// applyMiddleware wraps h with all registered middleware (last-in, first-executed).
func (r *Router) applyMiddleware(h Handler) Handler {
	for i := len(r.middleware) - 1; i >= 0; i-- {
		h = r.middleware[i](h)
	}
	return h
}

// ---------------------------------------------------------------------------
// Fluent builder for routing slips
// ---------------------------------------------------------------------------

// SlipBuilder constructs a []StepDef using a fluent API.
type SlipBuilder struct {
	steps []StepDef
}

// NewSlip creates a new SlipBuilder.
func NewSlip() *SlipBuilder { return &SlipBuilder{} }

// Step appends a step with optional parameters.
func (b *SlipBuilder) Step(name string, params ...map[string]any) *SlipBuilder {
	p := map[string]any{}
	if len(params) > 0 && params[0] != nil {
		p = params[0]
	}
	b.steps = append(b.steps, StepDef{Name: name, Params: p})
	return b
}

// Build returns the assembled slice of StepDef.
func (b *SlipBuilder) Build() []StepDef { return b.steps }

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

var (
	ErrValidationFailed = errors.New("validation failed")
	ErrSkipRemaining    = errors.New("skip remaining steps") // non-fatal stop signal
)
