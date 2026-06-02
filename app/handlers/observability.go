package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/raywall/routing-slip-pattern/slip"
)

// LogHandler writes an explicit structured log entry when the workflow needs a
// business or technical marker beyond the automatic execution logs.
type LogHandler struct{}

func (LogHandler) Name() string { return "log" }

func (LogHandler) Handle(ctx context.Context, msg *slip.Message, params map[string]any) (bool, error) {
	level := strings.ToLower(stringParam(params, "level", "info"))
	message := stringParam(params, "message", "routing-slip log")
	required := boolParam(params, "required", true)
	fields := stringListParam(params["fields"])

	attrs := []any{
		slog.String("message_id", msg.ID),
	}
	if msg.CorrelationID != "" {
		attrs = append(attrs, slog.String("correlation_id", msg.CorrelationID))
	}
	if msg.TraceID != "" {
		attrs = append(attrs, slog.String("trace_id", msg.TraceID))
	}
	if rawData, ok := params["data"]; ok {
		attrs = append(attrs, slog.Any("data", interpolateAny(rawData, msg)))
	}
	for _, field := range fields {
		if value, ok := msg.GetPath(field); ok {
			attrs = append(attrs, slog.Any(field, value))
			continue
		}
		if required {
			return false, fmt.Errorf("log: field %q not found", field)
		}
	}

	switch level {
	case "debug":
		slog.DebugContext(ctx, message, attrs...)
	case "warn", "warning":
		slog.WarnContext(ctx, message, attrs...)
	case "error":
		slog.ErrorContext(ctx, message, attrs...)
	default:
		slog.InfoContext(ctx, message, attrs...)
	}
	msg.Set("last_log_message", message)
	msg.Set("last_log_level", level)
	return true, nil
}

// DatadogMetricHandler emits a custom metric to the Datadog series API.
type DatadogMetricHandler struct {
	Client *http.Client
}

func (h DatadogMetricHandler) Name() string { return "datadog_metric" }

func (h DatadogMetricHandler) Handle(ctx context.Context, msg *slip.Message, params map[string]any) (bool, error) {
	required := boolParam(params, "required", true)
	if err := h.emit(ctx, msg, params); err != nil {
		if required {
			return false, err
		}
		msg.Set("datadog_metric_partial", true)
		return true, nil
	}
	return true, nil
}

func (h DatadogMetricHandler) emit(ctx context.Context, msg *slip.Message, params map[string]any) error {
	metric := stringParam(params, "metric", "")
	if strings.TrimSpace(metric) == "" {
		return fmt.Errorf("datadog_metric: metric is required")
	}
	apiKey := stringParam(params, "api_key", os.Getenv("DATADOG_API_KEY"))
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("datadog_metric: api_key is required")
	}
	apiURL := stringParam(params, "api_url", os.Getenv("DATADOG_API_URL"))
	if strings.TrimSpace(apiURL) == "" {
		apiURL = "https://api.datadoghq.com/api/v1/series"
	}
	value, err := numericParam(params["value"], 1)
	if err != nil {
		return fmt.Errorf("datadog_metric: value: %w", err)
	}
	timestamp := time.Now().Unix()
	if raw, ok := params["timestamp"]; ok {
		if parsed, err := numericParam(raw, float64(timestamp)); err == nil {
			timestamp = int64(parsed)
		}
	}

	tags := datadogTags(msg, params["tags"])
	metricType := stringParam(params, "type", "count")
	host := stringParam(params, "host", "")
	payload := map[string]any{
		"series": []map[string]any{
			{
				"metric": metric,
				"points": [][]any{{timestamp, value}},
				"type":   metricType,
				"tags":   tags,
			},
		},
	}
	if host != "" {
		payload["series"].([]map[string]any)[0]["host"] = host
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("datadog_metric: encode payload: %w", err)
	}

	timeout := durationParam(params, "timeout_ms", 2*time.Second)
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("datadog_metric: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", apiKey)

	client := h.Client
	if client == nil {
		client = &http.Client{Timeout: timeout + 200*time.Millisecond}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("datadog_metric: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("datadog_metric: datadog returned %s", resp.Status)
	}

	msg.Set("last_datadog_metric", metric)
	msg.Set("last_datadog_metric_at", time.Now().Format(time.RFC3339))
	return nil
}

func datadogTags(msg *slip.Message, raw any) []string {
	tags := []string{}
	if msg.CorrelationID != "" {
		tags = append(tags, "correlation_id:"+msg.CorrelationID)
	}
	switch typed := raw.(type) {
	case []any:
		for _, item := range typed {
			tag := fmt.Sprintf("%v", interpolateAny(item, msg))
			if strings.TrimSpace(tag) != "" {
				tags = append(tags, tag)
			}
		}
	case map[string]any:
		for key, value := range typed {
			tags = append(tags, key+":"+fmt.Sprintf("%v", interpolateAny(value, msg)))
		}
	}
	return tags
}

func numericParam(raw any, fallback float64) (float64, error) {
	if raw == nil {
		return fallback, nil
	}
	switch value := raw.(type) {
	case int:
		return float64(value), nil
	case int64:
		return float64(value), nil
	case float64:
		return value, nil
	case json.Number:
		return value.Float64()
	default:
		return 0, fmt.Errorf("must be numeric")
	}
}
