package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/raywall/routing-slip-pattern/slip"
)

type FilterArrayHandler struct{}

func (FilterArrayHandler) Name() string { return "filter_array" }

func (FilterArrayHandler) Handle(ctx context.Context, msg *slip.Message, params map[string]any) (bool, error) {
	source, _ := params["source"].(string)
	if strings.TrimSpace(source) == "" {
		source, _ = params["field"].(string)
	}
	if strings.TrimSpace(source) == "" {
		return false, fmt.Errorf("filter_array: source is required")
	}

	target, _ := params["target"].(string)
	if strings.TrimSpace(target) == "" {
		target = source
	}

	rawItems, ok := msg.GetPath(source)
	if !ok {
		return false, fmt.Errorf("filter_array: source %q not found", source)
	}
	items, ok := toAnySlice(rawItems)
	if !ok {
		return false, fmt.Errorf("filter_array: source %q must be an array", source)
	}

	filtered := make([]any, 0, len(items))
	for index, item := range items {
		matched, err := evaluateFilterArrayItem(msg, item, index, params)
		if err != nil {
			return false, fmt.Errorf("filter_array: item %d: %w", index, err)
		}
		if matched {
			filtered = append(filtered, item)
		}
	}

	if err := setPayloadPath(msg.Payload, target, filtered); err != nil {
		return false, fmt.Errorf("filter_array: %w", err)
	}
	msg.Set(target+"_filtered_count", len(filtered))
	msg.Set(target+"_removed_count", len(items)-len(filtered))
	return true, nil
}

func evaluateFilterArrayItem(msg *slip.Message, item any, index int, params map[string]any) (bool, error) {
	if expr, _ := params["expr"].(string); strings.TrimSpace(expr) != "" {
		return evaluateCELExpressionForPayload(filterArrayPayload(msg, item, index), msg.Headers, expr)
	}
	rawWhere, ok := params["where"]
	if !ok {
		return false, fmt.Errorf("where or expr is required")
	}
	itemMessage := slip.NewMessage(msg.ID, filterArrayPayload(msg, item, index))
	itemMessage.Headers = msg.Headers
	switch where := rawWhere.(type) {
	case map[string]any:
		matched, _, err := evaluateAssertConfig(itemMessage, where)
		return matched, err
	case []any:
		matched, _, err := evaluateAssertConfig(itemMessage, map[string]any{"all": where})
		return matched, err
	default:
		return false, fmt.Errorf("where must be an object or list")
	}
}

func filterArrayPayload(msg *slip.Message, item any, index int) map[string]any {
	payload := make(map[string]any, len(msg.Payload)+2)
	for key, value := range msg.Payload {
		payload[key] = value
	}
	payload["item"] = item
	payload["index"] = index
	return payload
}

func toAnySlice(value any) ([]any, bool) {
	switch typed := value.(type) {
	case []any:
		return typed, true
	case []map[string]any:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
		return items, true
	default:
		return nil, false
	}
}

func setPayloadPath(payload map[string]any, path string, value any) error {
	parts := strings.Split(path, ".")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return fmt.Errorf("target path is required")
	}
	current := payload
	for _, part := range parts[:len(parts)-1] {
		if strings.TrimSpace(part) == "" {
			return fmt.Errorf("invalid target path %q", path)
		}
		next, ok := current[part]
		if !ok {
			child := map[string]any{}
			current[part] = child
			current = child
			continue
		}
		child, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("target path %q crosses non-object field %q", path, part)
		}
		current = child
	}
	current[parts[len(parts)-1]] = value
	return nil
}
