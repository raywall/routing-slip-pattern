package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/raywall/routing-slip-pattern/app/slip"
)

type CELHandler struct{}

func (CELHandler) Name() string { return "cel" }

func (CELHandler) Handle(ctx context.Context, msg *slip.Message, params map[string]any) (bool, error) {
	expr, _ := params["expr"].(string)
	if strings.TrimSpace(expr) == "" {
		expr, _ = params["expression"].(string)
	}
	if strings.TrimSpace(expr) == "" {
		return false, fmt.Errorf("cel: expr is required")
	}

	matched, err := evaluateCELExpression(msg, expr)
	if err != nil {
		return false, fmt.Errorf("cel: %w", err)
	}

	target, _ := params["target"].(string)
	if strings.TrimSpace(target) == "" {
		target = "cel_passed"
	}
	msg.Set(target, matched)
	msg.Set("cel_passed", matched)

	if matched {
		return true, nil
	}

	onFalse, _ := params["on_false"].(string)
	onFalse = strings.ToLower(strings.TrimSpace(onFalse))
	if onFalse == "" {
		if _, ok := params["to"].(string); ok {
			onFalse = "jump"
		} else {
			onFalse = "error"
		}
	}

	switch onFalse {
	case "jump":
		target, _ := params["to"].(string)
		if strings.TrimSpace(target) == "" {
			return false, fmt.Errorf("cel: to is required when on_false is jump")
		}
		msg.Set("cel_jump_to", target)
		return true, nil
	case "continue":
		return true, nil
	case "stop":
		msg.Set("cel_stopped", true)
		return false, nil
	case "error", "fail":
		message, _ := params["message"].(string)
		if strings.TrimSpace(message) == "" {
			message = fmt.Sprintf("expression evaluated to false: %s", expr)
		}
		return false, fmt.Errorf("%s", message)
	default:
		return false, fmt.Errorf("cel: unsupported on_false value %q", onFalse)
	}
}

func (CELHandler) NextCursor(msg *slip.Message, step slip.StepDef, currentIndex int) (int, bool, error) {
	target, _ := msg.Get("cel_jump_to")
	targetName, _ := target.(string)
	if strings.TrimSpace(targetName) == "" {
		return 0, false, nil
	}
	index, ok := msg.FindStepIndex(targetName)
	if !ok {
		return 0, false, fmt.Errorf("target step %q not found", targetName)
	}
	if index <= currentIndex {
		return 0, false, fmt.Errorf("target step %q must be after current step", targetName)
	}
	msg.Set("jumped_to", targetName)
	msg.Set("jumped_from_cursor", currentIndex)
	msg.Set("jumped_to_cursor", index)
	msg.Set("cel_jump_to", "")
	return index, true, nil
}

func evaluateCELExpression(msg *slip.Message, expr string) (bool, error) {
	return evaluateCELExpressionForPayload(msg.Payload, msg.Headers, expr)
}

func evaluateCELExpressionForPayload(payload map[string]any, headers map[string]string, expr string) (bool, error) {
	options := []cel.EnvOption{
		cel.Variable("payload", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("headers", cel.MapType(cel.StringType, cel.StringType)),
	}
	activation := map[string]any{
		"payload": payload,
		"headers": headers,
	}

	for key, value := range payload {
		if key == "payload" || key == "headers" || !isCELIdentifier(key) {
			continue
		}
		options = append(options, cel.Variable(key, cel.DynType))
		activation[key] = value
	}

	env, err := cel.NewEnv(options...)
	if err != nil {
		return false, err
	}
	ast, issues := env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return false, issues.Err()
	}
	program, err := env.Program(ast)
	if err != nil {
		return false, err
	}
	out, _, err := program.Eval(activation)
	if err != nil {
		return false, err
	}
	value, ok := out.Value().(bool)
	if !ok {
		return false, fmt.Errorf("expression must return bool, got %T", out.Value())
	}
	return value, nil
}

func isCELIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for index, char := range value {
		if index == 0 {
			if char != '_' && (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') {
				return false
			}
			continue
		}
		if char != '_' && (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}
