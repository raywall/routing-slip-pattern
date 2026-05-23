package handlers

import (
	"context"
	"strings"
	"testing"

	"github.com/raywall/routing-slip-pattern/slip"
)

func TestCELHandlerPassesWithTopLevelPayloadVariables(t *testing.T) {
	msg := slip.NewMessage("MSG", map[string]any{
		"pedido": map[string]any{"status": "APROVADO"},
		"itens":  []any{map[string]any{"sku": "SKU-1"}},
	})

	proceed, err := CELHandler{}.Handle(context.Background(), msg, map[string]any{
		"expr": "pedido.status == 'APROVADO' && size(itens) > 0",
	})
	if err != nil {
		t.Fatalf("cel handle: %v", err)
	}
	if !proceed {
		t.Fatal("expected workflow to proceed")
	}
	value, _ := msg.Get("cel_passed")
	if value != true {
		t.Fatalf("cel_passed = %v", value)
	}
}

func TestCELHandlerFailsWhenRequiredExpressionIsFalse(t *testing.T) {
	msg := slip.NewMessage("MSG", map[string]any{
		"pedido": map[string]any{"status": "CANCELADO"},
	})

	_, err := CELHandler{}.Handle(context.Background(), msg, map[string]any{
		"expr":    "pedido.status == 'APROVADO'",
		"message": "Pedido nao pode prosseguir.",
	})
	if err == nil {
		t.Fatal("expected cel to fail")
	}
	if !strings.Contains(err.Error(), "Pedido nao pode prosseguir") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCELHandlerJumpsWhenExpressionIsFalse(t *testing.T) {
	msg := slip.NewMessage("MSG", map[string]any{
		"pedido": map[string]any{"total": 0},
	})
	msg.AttachSlip([]slip.StepDef{
		{Name: "cel", Params: map[string]any{"expr": "pedido.total > 0", "on_false": "jump", "to": "revisar"}},
		{Name: "enrich"},
		{ID: "revisar", Name: "audit"},
	})
	step := slip.StepDef{Name: "cel", Params: map[string]any{"expr": "pedido.total > 0", "on_false": "jump", "to": "revisar"}}

	proceed, err := CELHandler{}.Handle(context.Background(), msg, step.Params)
	if err != nil {
		t.Fatalf("cel handle: %v", err)
	}
	if !proceed {
		t.Fatal("expected workflow to proceed before cursor control")
	}
	cursor, changed, err := CELHandler{}.NextCursor(msg, step, 0)
	if err != nil {
		t.Fatalf("cel next cursor: %v", err)
	}
	if !changed || cursor != 2 {
		t.Fatalf("cursor = %d changed = %v", cursor, changed)
	}
}

func TestCELHandlerCanContinueWhenExpressionIsFalse(t *testing.T) {
	msg := slip.NewMessage("MSG", map[string]any{
		"pedido": map[string]any{"status": "PENDENTE"},
	})

	proceed, err := CELHandler{}.Handle(context.Background(), msg, map[string]any{
		"expr":     "pedido.status == 'APROVADO'",
		"on_false": "continue",
		"target":   "pedido_aprovado",
	})
	if err != nil {
		t.Fatalf("cel handle: %v", err)
	}
	if !proceed {
		t.Fatal("expected workflow to continue")
	}
	value, _ := msg.Get("pedido_aprovado")
	if value != false {
		t.Fatalf("pedido_aprovado = %v", value)
	}
}
