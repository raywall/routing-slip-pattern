package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/raywall/routing-slip-pattern/slip"
)

type ArrayTransformHandler struct{}

func (ArrayTransformHandler) Name() string { return "array_transform" }

func (ArrayTransformHandler) Handle(ctx context.Context, msg *slip.Message, params map[string]any) (bool, error) {
	source, _ := params["source"].(string)
	if strings.TrimSpace(source) == "" {
		return false, fmt.Errorf("array_transform: source is required")
	}
	target, _ := params["target"].(string)
	if strings.TrimSpace(target) == "" {
		target = source
	}

	rawItems, ok := msg.GetPath(source)
	if !ok {
		return false, fmt.Errorf("array_transform: source %q not found", source)
	}
	items, ok := toAnySlice(rawItems)
	if !ok {
		return false, fmt.Errorf("array_transform: source %q must be an array", source)
	}

	transformed, err := transformArrayItems(msg, items, params, nil)
	if err != nil {
		return false, err
	}
	if err := setPayloadPath(msg.Payload, target, transformed); err != nil {
		return false, fmt.Errorf("array_transform: %w", err)
	}
	msg.Set(target+"_transformed_count", len(transformed))
	return true, nil
}

func transformArrayItems(msg *slip.Message, items []any, params map[string]any, parent any) ([]any, error) {
	filtered := make([]any, 0, len(items))
	for index, item := range items {
		working := copyTransformValue(item)
		payload := arrayTransformPayload(msg, working, index, parent)

		keep, err := evaluateArrayTransformFilters(msg, payload, params)
		if err != nil {
			return nil, fmt.Errorf("array_transform: item %d: %w", index, err)
		}
		if !keep {
			continue
		}

		itemMap, _ := working.(map[string]any)
		if itemMap != nil {
			if err := applyArrayTransformUpdates(msg, itemMap, payload, params); err != nil {
				return nil, fmt.Errorf("array_transform: item %d: %w", index, err)
			}
			if err := applyNestedArrayTransforms(msg, itemMap, params); err != nil {
				return nil, fmt.Errorf("array_transform: item %d: %w", index, err)
			}
		}
		filtered = append(filtered, working)
	}
	return filtered, nil
}

func evaluateArrayTransformFilters(msg *slip.Message, payload map[string]any, params map[string]any) (bool, error) {
	rawFilters, ok := params["filters"]
	if !ok {
		rawFilters, ok = params["where"]
	}
	if !ok {
		return true, nil
	}
	return evaluateArrayTransformCondition(msg, payload, rawFilters)
}

func applyArrayTransformUpdates(msg *slip.Message, item map[string]any, payload map[string]any, params map[string]any) error {
	rawUpdates, ok := params["updates"]
	if !ok {
		return nil
	}
	updates, ok := rawUpdates.([]any)
	if !ok {
		return fmt.Errorf("updates must be a list")
	}
	for index, rawUpdate := range updates {
		update, ok := rawUpdate.(map[string]any)
		if !ok {
			return fmt.Errorf("updates[%d] must be an object", index)
		}
		if rawWhen, ok := update["when"]; ok {
			matched, err := evaluateArrayTransformCondition(msg, payload, rawWhen)
			if err != nil {
				return fmt.Errorf("updates[%d].when: %w", index, err)
			}
			if !matched {
				continue
			}
		}
		rawSet, ok := update["set"].(map[string]any)
		if !ok {
			return fmt.Errorf("updates[%d].set must be an object", index)
		}
		for path, value := range rawSet {
			resolved := resolveTransformValue(payload, value)
			if err := setPayloadPath(item, path, resolved); err != nil {
				return err
			}
		}
		payload["item"] = item
	}
	return nil
}

func applyNestedArrayTransforms(msg *slip.Message, item map[string]any, params map[string]any) error {
	rawNested, ok := params["nested"]
	if !ok {
		return nil
	}
	nestedTransforms, ok := rawNested.([]any)
	if !ok {
		return fmt.Errorf("nested must be a list")
	}
	for index, rawNestedTransform := range nestedTransforms {
		nestedTransform, ok := rawNestedTransform.(map[string]any)
		if !ok {
			return fmt.Errorf("nested[%d] must be an object", index)
		}
		source, _ := nestedTransform["source"].(string)
		if strings.TrimSpace(source) == "" {
			source, _ = nestedTransform["field"].(string)
		}
		if strings.TrimSpace(source) == "" {
			return fmt.Errorf("nested[%d].source is required", index)
		}
		target, _ := nestedTransform["target"].(string)
		if strings.TrimSpace(target) == "" {
			target = source
		}
		rawItems, ok := getMapPath(item, source)
		if !ok {
			continue
		}
		items, ok := toAnySlice(rawItems)
		if !ok {
			return fmt.Errorf("nested[%d].source %q must be an array", index, source)
		}
		transformed, err := transformArrayItems(msg, items, nestedTransform, item)
		if err != nil {
			return err
		}
		if err := setPayloadPath(item, target, transformed); err != nil {
			return err
		}
	}
	return nil
}

func evaluateArrayTransformCondition(msg *slip.Message, payload map[string]any, condition any) (bool, error) {
	switch typed := condition.(type) {
	case string:
		return evaluateCELExpressionForPayload(arrayTransformRuntimePayload(payload), msg.Headers, typed)
	case map[string]any:
		if expr, _ := typed["expr"].(string); strings.TrimSpace(expr) != "" {
			return evaluateCELExpressionForPayload(arrayTransformRuntimePayload(payload), msg.Headers, expr)
		}
		itemMessage := slip.NewMessage(msg.ID, arrayTransformRuntimePayload(payload))
		itemMessage.Headers = msg.Headers
		matched, _, err := evaluateAssertConfig(itemMessage, typed)
		return matched, err
	case []any:
		itemMessage := slip.NewMessage(msg.ID, arrayTransformRuntimePayload(payload))
		itemMessage.Headers = msg.Headers
		matched, _, err := evaluateAssertConfig(itemMessage, map[string]any{"all": typed})
		return matched, err
	default:
		return false, fmt.Errorf("condition must be an object, list or expression")
	}
}

func arrayTransformPayload(msg *slip.Message, item any, index int, parent any) map[string]any {
	payload := make(map[string]any, len(msg.Payload)+5)
	for key, value := range msg.Payload {
		payload[key] = value
	}
	payload["item"] = item
	payload["index"] = index
	payload["parent"] = parent
	payload["today"] = time.Now().Format("2006-01-02")
	payload["end_of_current_month_plus_2"] = endOfCurrentMonthPlus(2).Format("2006-01-02")
	return payload
}

func arrayTransformRuntimePayload(payload map[string]any) map[string]any {
	return payload
}

func resolveTransformValue(payload map[string]any, value any) any {
	if config, ok := value.(map[string]any); ok {
		if from, ok := config["from"].(string); ok {
			if resolved, exists := getMapPath(payload, from); exists {
				return resolved
			}
			return nil
		}
	}
	return value
}

func getMapPath(values map[string]any, path string) (any, bool) {
	if strings.TrimSpace(path) == "" {
		return nil, false
	}
	var current any = values
	for _, part := range strings.Split(path, ".") {
		currentMap, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = currentMap[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func copyTransformValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		copied := make(map[string]any, len(typed))
		for key, item := range typed {
			copied[key] = copyTransformValue(item)
		}
		return copied
	case []any:
		copied := make([]any, len(typed))
		for index, item := range typed {
			copied[index] = copyTransformValue(item)
		}
		return copied
	default:
		return value
	}
}

func endOfCurrentMonthPlus(months int) time.Time {
	now := time.Now()
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	return firstOfMonth.AddDate(0, months+1, 0).AddDate(0, 0, -1)
}
