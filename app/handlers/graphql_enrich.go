package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/raywall/routing-slip-pattern/slip"
)

// GraphQLEnrichmentHandler enriches the payload with data returned by a GraphQL
// endpoint such as go-graphql-connector.
type GraphQLEnrichmentHandler struct {
	DefaultEndpoint string
	Client          *http.Client
}

func (h GraphQLEnrichmentHandler) Name() string { return "graphql_enrich" }

func (h GraphQLEnrichmentHandler) Handle(ctx context.Context, msg *slip.Message, params map[string]any) (bool, error) {
	endpoint := stringParam(params, "endpoint", h.DefaultEndpoint)
	query := stringParam(params, "query", "")
	target := stringParam(params, "target", "external_data")
	resultPath := stringParam(params, "result_path", "")
	required := boolParam(params, "required", true)
	timeout := durationParam(params, "timeout_ms", 2*time.Second)
	if endpoint == "" || query == "" {
		return true, fmt.Errorf("graphql_enrich: endpoint and query are required")
	}

	variables := map[string]any{}
	if raw, ok := params["variables"].(map[string]any); ok {
		for k, v := range raw {
			variables[k] = interpolateValue(v, msg)
		}
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	body, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		return true, err
	}

	client := h.Client
	if client == nil {
		client = &http.Client{Timeout: timeout + 200*time.Millisecond}
	}
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return true, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		if required {
			return false, fmt.Errorf("graphql_enrich: %w", err)
		}
		msg.Set(target+"_partial", true)
		return true, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("graphql_enrich: endpoint returned %s", resp.Status)
		if required {
			return false, err
		}
		msg.Set(target+"_partial", true)
		return true, nil
	}

	var response struct {
		Data   map[string]any   `json:"data"`
		Errors []map[string]any `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return false, fmt.Errorf("graphql_enrich: decode response: %w", err)
	}
	if len(response.Errors) > 0 && required {
		return false, fmt.Errorf("graphql_enrich: graphql errors: %v", response.Errors)
	}

	value := any(response.Data)
	if resultPath != "" {
		if selected, ok := selectPath(response.Data, resultPath); ok {
			value = selected
		} else if required {
			return false, fmt.Errorf("graphql_enrich: result_path %q not found", resultPath)
		}
	}
	msg.Set(target, value)
	msg.Set(target+"_enriched_at", time.Now().Format(time.RFC3339))
	return true, nil
}

func interpolateValue(value any, msg *slip.Message) any {
	text, ok := value.(string)
	if !ok {
		return value
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

func selectPath(values map[string]any, path string) (any, bool) {
	var current any = values
	for _, part := range strings.Split(path, ".") {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func stringParam(params map[string]any, key, fallback string) string {
	value, ok := params[key].(string)
	if !ok || value == "" {
		return fallback
	}
	return value
}

func boolParam(params map[string]any, key string, fallback bool) bool {
	value, ok := params[key].(bool)
	if !ok {
		return fallback
	}
	return value
}

func durationParam(params map[string]any, key string, fallback time.Duration) time.Duration {
	raw, ok := params[key]
	if !ok {
		return fallback
	}
	switch value := raw.(type) {
	case int:
		if value > 0 {
			return time.Duration(value) * time.Millisecond
		}
	case int64:
		if value > 0 {
			return time.Duration(value) * time.Millisecond
		}
	case float64:
		if value > 0 {
			return time.Duration(value) * time.Millisecond
		}
	}
	return fallback
}
