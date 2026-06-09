package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/raywall/routing-slip-pattern/app/slip"
)

// RESTCallHandler calls an HTTP endpoint and stores the decoded JSON response
// in the message payload. It is useful for domain actions such as invoking a
// Lambda URL, dispatching an expedition request, or updating inventory.
type RESTCallHandler struct {
	Client *http.Client
}

func (h RESTCallHandler) Name() string { return "rest_call" }

func (h RESTCallHandler) Handle(ctx context.Context, msg *slip.Message, params map[string]any) (bool, error) {
	baseURL := stringParam(params, "base_url", "")
	endpoint := stringParam(params, "endpoint", "")
	method := strings.ToUpper(stringParam(params, "method", http.MethodGet))
	target := stringParam(params, "target", "http_response")
	resultPath := stringParam(params, "result_path", "")
	timeout := durationParam(params, "timeout_ms", 2*time.Second)
	required := boolParam(params, "required", true)
	if baseURL == "" || endpoint == "" {
		return false, fmt.Errorf("rest_call: base_url and endpoint are required")
	}

	endpoint = fmt.Sprintf("%v", interpolateAny(endpoint, msg))
	var bodyReader *bytes.Reader
	if rawBody, ok := params["body"]; ok && method != http.MethodGet && method != http.MethodHead {
		body := interpolateAny(rawBody, msg)
		data, err := json.Marshal(body)
		if err != nil {
			return false, fmt.Errorf("rest_call: encode body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, method, strings.TrimRight(baseURL, "/")+endpoint, bodyReader)
	if err != nil {
		return false, err
	}
	req.Header.Set("Accept", "application/json")
	if traceparent := msg.Headers["traceparent"]; traceparent != "" {
		req.Header.Set("traceparent", traceparent)
	}
	if msg.TraceID != "" {
		req.Header.Set("X-Trace-ID", msg.TraceID)
	}
	if msg.CorrelationID != "" {
		req.Header.Set("X-Correlation-ID", msg.CorrelationID)
	}
	if bodyReader.Len() > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if headers, ok := params["headers"].(map[string]any); ok {
		for key, value := range headers {
			req.Header.Set(key, fmt.Sprintf("%v", interpolateAny(value, msg)))
		}
	}

	client := h.Client
	if client == nil {
		client = &http.Client{Timeout: timeout + 200*time.Millisecond}
	}
	resp, err := client.Do(req)
	if err != nil {
		if required {
			return false, fmt.Errorf("rest_call: %w", err)
		}
		msg.Set(target+"_partial", true)
		return true, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("rest_call: endpoint returned %s", resp.Status)
		if required {
			return false, err
		}
		msg.Set(target+"_partial", true)
		return true, nil
	}

	var response map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return false, fmt.Errorf("rest_call: decode response: %w", err)
	}
	value := any(response)
	if resultPath != "" {
		if selected, ok := selectPath(response, resultPath); ok {
			value = selected
		} else if required {
			return false, fmt.Errorf("rest_call: result_path %q not found", resultPath)
		}
	}
	msg.Set(target, value)
	msg.Set(target+"_called_at", time.Now().Format(time.RFC3339))
	return true, nil
}

func interpolateAny(value any, msg *slip.Message) any {
	switch typed := value.(type) {
	case string:
		return interpolateString(typed, msg)
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			result[key] = interpolateAny(item, msg)
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for i, item := range typed {
			result[i] = interpolateAny(item, msg)
		}
		return result
	default:
		return value
	}
}

func interpolateString(text string, msg *slip.Message) any {
	if strings.HasPrefix(text, "{") && strings.HasSuffix(text, "}") && strings.Count(text, "{") == 1 {
		key := strings.TrimSuffix(strings.TrimPrefix(text, "{"), "}")
		if replacement, ok := msg.GetPath(key); ok {
			return replacement
		}
	}
	for strings.Contains(text, "{") {
		start := strings.Index(text, "{")
		end := strings.Index(text, "}")
		if start == -1 || end == -1 || end < start {
			break
		}
		key := text[start+1 : end]
		replacement, _ := msg.GetPath(key)
		text = text[:start] + fmt.Sprintf("%v", replacement) + text[end+1:]
	}
	return text
}
