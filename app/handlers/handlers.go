// Package handlers provides a set of ready-to-use routing slip handlers.
package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/raywall/routing-slip-pattern/slip"
)

// ---------------------------------------------------------------------------
// ValidationHandler
// ---------------------------------------------------------------------------

// ValidationHandler checks that required fields are present and non-empty.
//
// Params:
//
//	"required" []string  – list of payload keys that must exist
//	"stop_on_failure" bool – if true (default), returns error; else just logs
type ValidationHandler struct{}

func (ValidationHandler) Name() string { return "validate" }

func (ValidationHandler) Handle(ctx context.Context, msg *slip.Message, params map[string]any) (bool, error) {
	required, _ := params["required"].([]string)
	stopOnFailure := true
	if v, ok := params["stop_on_failure"].(bool); ok {
		stopOnFailure = v
	}

	var missing []string
	for _, key := range required {
		v, ok := msg.Get(key)
		if !ok || v == nil || fmt.Sprintf("%v", v) == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		err := fmt.Errorf("%w: missing fields: %s", slip.ErrValidationFailed, strings.Join(missing, ", "))
		msg.Set("validation_error", err.Error())
		if stopOnFailure {
			return false, err
		}
		slog.Warn("validation: fields missing (continuing)", slog.String("fields", strings.Join(missing, ", ")))
	} else {
		msg.Set("validation_passed", true)
	}
	return true, nil
}

// ---------------------------------------------------------------------------
// EnrichmentHandler
// ---------------------------------------------------------------------------

// EnrichmentHandler merges static or computed data into the payload.
//
// Params:
//
//	"data" map[string]any – key/value pairs to merge into the payload
//	"prefix" string       – optional prefix for injected keys
type EnrichmentHandler struct{}

func (EnrichmentHandler) Name() string { return "enrich" }

func (EnrichmentHandler) Handle(ctx context.Context, msg *slip.Message, params map[string]any) (bool, error) {
	data, _ := params["data"].(map[string]any)
	prefix, _ := params["prefix"].(string)

	for k, v := range data {
		key := prefix + k
		msg.Set(key, v)
	}
	// Always stamp enrichment time
	msg.Set(prefix+"enriched_at", time.Now().Format(time.RFC3339))
	return true, nil
}

// ---------------------------------------------------------------------------
// TransformHandler
// ---------------------------------------------------------------------------

// TransformHandler applies a named transformation to a payload field.
//
// Params:
//
//	"field"      string – the key to transform
//	"operation"  string – one of: "uppercase", "lowercase", "trim", "prefix:<val>", "suffix:<val>"
//	"target"     string – destination key (defaults to field)
type TransformHandler struct{}

func (TransformHandler) Name() string { return "transform" }

func (TransformHandler) Handle(ctx context.Context, msg *slip.Message, params map[string]any) (bool, error) {
	field, _ := params["field"].(string)
	operation, _ := params["operation"].(string)
	target, _ := params["target"].(string)
	if target == "" {
		target = field
	}

	raw, ok := msg.GetPath(field)
	if !ok {
		return true, fmt.Errorf("transform: field %q not found in payload", field)
	}
	value := fmt.Sprintf("%v", raw)

	switch {
	case operation == "uppercase":
		value = strings.ToUpper(value)
	case operation == "lowercase":
		value = strings.ToLower(value)
	case operation == "trim":
		value = strings.TrimSpace(value)
	case strings.HasPrefix(operation, "prefix:"):
		value = strings.TrimPrefix(operation, "prefix:") + value
	case strings.HasPrefix(operation, "suffix:"):
		value = value + strings.TrimPrefix(operation, "suffix:")
	default:
		return true, fmt.Errorf("transform: unknown operation %q", operation)
	}

	msg.Set(target, value)
	return true, nil
}

// ---------------------------------------------------------------------------
// ConditionGate
// ---------------------------------------------------------------------------

// ConditionGate stops processing if a payload field does not match the expected value.
//
// Params:
//
//	"field"    string – the key to check
//	"equals"   any    – stop if the field value does NOT equal this
//	"not_equals" any  – stop if the field value EQUALS this
type ConditionGate struct{}

func (ConditionGate) Name() string { return "condition" }

func (ConditionGate) Handle(ctx context.Context, msg *slip.Message, params map[string]any) (bool, error) {
	field, _ := params["field"].(string)
	val, _ := msg.GetPath(field)
	current := fmt.Sprintf("%v", val)

	if expected, ok := params["equals"]; ok {
		if current != fmt.Sprintf("%v", expected) {
			slog.Info("condition gate: stopping workflow",
				slog.String("field", field),
				slog.String("expected", fmt.Sprintf("%v", expected)),
				slog.String("got", current),
			)
			msg.Set("gate_stopped", true)
			return false, nil // graceful stop
		}
	}

	if excluded, ok := params["not_equals"]; ok {
		if current == fmt.Sprintf("%v", excluded) {
			slog.Info("condition gate: stopping workflow (not_equals matched)",
				slog.String("field", field),
				slog.String("value", current),
			)
			msg.Set("gate_stopped", true)
			return false, nil
		}
	}

	return true, nil
}

// ---------------------------------------------------------------------------
// NotificationHandler (simulated)
// ---------------------------------------------------------------------------

// NotificationHandler simulates sending a notification (email, webhook, etc.).
//
// Params:
//
//	"channel"   string – "email" | "webhook" | "slack"
//	"template"  string – message template; use {field} to interpolate payload values
//	"recipient" string – destination address / URL
type NotificationHandler struct {
	// Send is the actual delivery function; defaults to a no-op logger.
	Send func(channel, recipient, body string) error
}

func (n *NotificationHandler) Name() string { return "notify" }

func (n *NotificationHandler) Handle(ctx context.Context, msg *slip.Message, params map[string]any) (bool, error) {
	channel, _ := params["channel"].(string)
	template, _ := params["template"].(string)
	recipient, _ := params["recipient"].(string)

	body := interpolate(template, msg)

	deliver := n.Send
	if deliver == nil {
		deliver = defaultSend
	}

	if err := deliver(channel, recipient, body); err != nil {
		return false, fmt.Errorf("notify: %w", err)
	}

	msg.Set("notification_sent", true)
	msg.Set("notification_channel", channel)
	return true, nil
}

func defaultSend(channel, recipient, body string) error {
	slog.Info("notification dispatched",
		slog.String("channel", channel),
		slog.String("recipient", recipient),
		slog.String("body", body),
	)
	return nil
}

// interpolate replaces {key} tokens in the template with payload values.
func interpolate(template string, msg *slip.Message) string {
	result := template
	// Simple O(n) scan; replace known keys.
	msg.Headers["_interpolate"] = "1" // dummy to iterate below
	// We iterate the payload directly by peeking at known replacements.
	// In real code use text/template for full power.
	for strings.Contains(result, "{") {
		start := strings.Index(result, "{")
		end := strings.Index(result, "}")
		if start == -1 || end == -1 || end < start {
			break
		}
		key := result[start+1 : end]
		val, _ := msg.Get(key)
		result = result[:start] + fmt.Sprintf("%v", val) + result[end+1:]
	}
	delete(msg.Headers, "_interpolate")
	return result
}

// ---------------------------------------------------------------------------
// AuditHandler
// ---------------------------------------------------------------------------

// AuditHandler writes a structured audit record to slog.
//
// Params:
//
//	"event" string – event name to record (defaults to "audit")
//	"fields" []string – payload keys to include in the audit record
type AuditHandler struct{}

func (AuditHandler) Name() string { return "audit" }

func (AuditHandler) Handle(ctx context.Context, msg *slip.Message, params map[string]any) (bool, error) {
	event, _ := params["event"].(string)
	if event == "" {
		event = "audit"
	}
	fields, _ := params["fields"].([]string)

	attrs := []any{
		slog.String("event", event),
		slog.String("message_id", msg.ID),
		slog.Time("timestamp", time.Now()),
	}
	for _, f := range fields {
		v, _ := msg.GetPath(f)
		attrs = append(attrs, slog.Any(f, v))
	}
	slog.Info("audit", attrs...)
	return true, nil
}
