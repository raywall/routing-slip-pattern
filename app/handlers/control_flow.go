package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/raywall/routing-slip-pattern/slip"
)

type ComputeHandler struct{}

func (ComputeHandler) Name() string { return "compute" }

func (ComputeHandler) Handle(ctx context.Context, msg *slip.Message, params map[string]any) (bool, error) {
	target, _ := params["target"].(string)
	if strings.TrimSpace(target) == "" {
		return false, fmt.Errorf("compute: target is required")
	}
	valueConfig, ok := params["value"].(map[string]any)
	if !ok {
		return false, fmt.Errorf("compute: value must be an object")
	}
	value, err := evaluateValueConfig(msg, valueConfig)
	if err != nil {
		return false, fmt.Errorf("compute: %w", err)
	}
	msg.Set(target, value)
	return true, nil
}

type AssertHandler struct{}

func (AssertHandler) Name() string { return "assert" }

func (AssertHandler) Handle(ctx context.Context, msg *slip.Message, params map[string]any) (bool, error) {
	matched, failures, err := evaluateAssertConfig(msg, params)
	if err != nil {
		return false, fmt.Errorf("assert: %w", err)
	}
	msg.Set("assert_passed", matched)
	if matched {
		return true, nil
	}
	message, _ := params["message"].(string)
	if strings.TrimSpace(message) == "" {
		message = "assertion failed"
	}
	if len(failures) > 0 {
		msg.Set("assert_failures", failures)
		return false, fmt.Errorf("%s: %s", message, strings.Join(failures, "; "))
	}
	return false, fmt.Errorf("%s", message)
}

type JumpIfHandler struct{}

func (JumpIfHandler) Name() string { return "jump_if" }

func (JumpIfHandler) Handle(ctx context.Context, msg *slip.Message, params map[string]any) (bool, error) {
	matched, err := evaluateConditionConfig(msg, params)
	if err != nil {
		return false, fmt.Errorf("jump_if: %w", err)
	}
	msg.Set("jump_if_matched", matched)
	return true, nil
}

func (JumpIfHandler) NextCursor(msg *slip.Message, step slip.StepDef, currentIndex int) (int, bool, error) {
	matched, _ := msg.Get("jump_if_matched")
	if matched != true {
		return 0, false, nil
	}
	target, _ := step.Params["to"].(string)
	index, ok := msg.FindStepIndex(target)
	if !ok {
		return 0, false, fmt.Errorf("target step %q not found", target)
	}
	if index <= currentIndex {
		return 0, false, fmt.Errorf("target step %q must be after current step", target)
	}
	msg.Set("jumped_to", target)
	msg.Set("jumped_from_cursor", currentIndex)
	msg.Set("jumped_to_cursor", index)
	return index, true, nil
}

func evaluateAssertConfig(msg *slip.Message, params map[string]any) (bool, []string, error) {
	if rawAll, ok := params["all"]; ok {
		conditions, err := conditionList(rawAll)
		if err != nil {
			return false, nil, err
		}
		failures := make([]string, 0)
		for index, condition := range conditions {
			matched, err := evaluateConditionConfig(msg, condition)
			if err != nil {
				failures = append(failures, fmt.Sprintf("all[%d]: %v", index, err))
				continue
			}
			if !matched {
				failures = append(failures, fmt.Sprintf("all[%d]: condition not satisfied", index))
			}
		}
		return len(failures) == 0, failures, nil
	}

	if rawAny, ok := params["any"]; ok {
		conditions, err := conditionList(rawAny)
		if err != nil {
			return false, nil, err
		}
		failures := make([]string, 0)
		for index, condition := range conditions {
			matched, err := evaluateConditionConfig(msg, condition)
			if err != nil {
				failures = append(failures, fmt.Sprintf("any[%d]: %v", index, err))
				continue
			}
			if matched {
				return true, nil, nil
			}
			failures = append(failures, fmt.Sprintf("any[%d]: condition not satisfied", index))
		}
		return false, failures, nil
	}

	matched, err := evaluateConditionConfig(msg, params)
	if err != nil {
		return false, nil, err
	}
	if matched {
		return true, nil, nil
	}
	return false, []string{"condition not satisfied"}, nil
}

func conditionList(value any) ([]map[string]any, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("conditions must be a list")
	}
	conditions := make([]map[string]any, 0, len(items))
	for index, item := range items {
		condition, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("condition %d must be an object", index)
		}
		conditions = append(conditions, condition)
	}
	return conditions, nil
}

func evaluateValueConfig(msg *slip.Message, config map[string]any) (any, error) {
	if value, ok := config["literal"]; ok {
		return value, nil
	}
	if field, ok := config["field"].(string); ok && len(config) == 1 {
		value, exists := msg.GetPath(field)
		if !exists {
			return nil, fmt.Errorf("field %q not found", field)
		}
		return value, nil
	}
	if field, ok := config["count"].(string); ok {
		value, exists := msg.GetPath(field)
		if !exists {
			return nil, fmt.Errorf("field %q not found", field)
		}
		count, ok := collectionLength(value)
		if !ok {
			return nil, fmt.Errorf("field %q is not countable", field)
		}
		return count, nil
	}
	if _, ok := config["exists"]; ok {
		return evaluateConditionConfig(msg, config)
	}
	if _, ok := config["field"]; ok {
		return evaluateConditionConfig(msg, config)
	}
	return nil, fmt.Errorf("unsupported value config")
}

func evaluateConditionConfig(msg *slip.Message, config map[string]any) (bool, error) {
	if rawExists, ok := config["exists"]; ok {
		field, ok := rawExists.(string)
		if !ok {
			field, _ = config["field"].(string)
		}
		if strings.TrimSpace(field) == "" {
			return false, fmt.Errorf("exists must be a field path")
		}
		_, exists := msg.GetPath(field)
		if expected, ok := rawExists.(bool); ok {
			return exists == expected, nil
		}
		return exists, nil
	}

	field, _ := config["field"].(string)
	if strings.TrimSpace(field) == "" {
		return false, fmt.Errorf("field is required")
	}
	left, exists := msg.GetPath(field)
	if !exists {
		return false, fmt.Errorf("field %q not found", field)
	}

	switch {
	case hasKey(config, "equals"):
		return valuesEqual(left, config["equals"]), nil
	case hasKey(config, "not_equals"):
		return !valuesEqual(left, config["not_equals"]), nil
	case hasKey(config, "less_than"):
		return compareNumbers(left, config["less_than"], func(a, b float64) bool { return a < b })
	case hasKey(config, "less_than_or_equal"):
		return compareNumbers(left, config["less_than_or_equal"], func(a, b float64) bool { return a <= b })
	case hasKey(config, "greater_than"):
		return compareNumbers(left, config["greater_than"], func(a, b float64) bool { return a > b })
	case hasKey(config, "greater_than_or_equal"):
		return compareNumbers(left, config["greater_than_or_equal"], func(a, b float64) bool { return a >= b })
	case hasKey(config, "min_items"):
		count, ok := collectionLength(left)
		if !ok {
			return false, fmt.Errorf("field %q is not countable", field)
		}
		min, err := toFloat(config["min_items"])
		if err != nil {
			return false, err
		}
		return float64(count) >= min, nil
	case hasKey(config, "max_items"):
		count, ok := collectionLength(left)
		if !ok {
			return false, fmt.Errorf("field %q is not countable", field)
		}
		max, err := toFloat(config["max_items"])
		if err != nil {
			return false, err
		}
		return float64(count) <= max, nil
	default:
		return false, fmt.Errorf("no supported comparison configured")
	}
}

func hasKey(values map[string]any, key string) bool {
	_, ok := values[key]
	return ok
}

func valuesEqual(left, right any) bool {
	if leftNumber, err := toFloat(left); err == nil {
		if rightNumber, err := toFloat(right); err == nil {
			return leftNumber == rightNumber
		}
	}
	return fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right) || reflect.DeepEqual(left, right)
}

func compareNumbers(left, right any, compare func(float64, float64) bool) (bool, error) {
	leftNumber, err := toFloat(left)
	if err != nil {
		return false, err
	}
	rightNumber, err := toFloat(right)
	if err != nil {
		return false, err
	}
	return compare(leftNumber, rightNumber), nil
}

func toFloat(value any) (float64, error) {
	switch typed := value.(type) {
	case int:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case float64:
		return typed, nil
	case float32:
		return float64(typed), nil
	case json.Number:
		return typed.Float64()
	case string:
		return strconv.ParseFloat(strings.TrimSpace(typed), 64)
	default:
		return 0, fmt.Errorf("value %v is not numeric", value)
	}
}

func collectionLength(value any) (int, bool) {
	switch typed := value.(type) {
	case []any:
		return len(typed), true
	case map[string]any:
		return len(typed), true
	case string:
		return len(typed), true
	default:
		rv := reflect.ValueOf(value)
		switch rv.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
			return rv.Len(), true
		default:
			return 0, false
		}
	}
}
