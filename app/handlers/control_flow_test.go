package handlers

import (
	"context"
	"testing"

	"github.com/raywall/routing-slip-pattern/slip"
)

func TestComputeLessThanOrEqualWithIndexedPath(t *testing.T) {
	msg := slip.NewMessage("MSG", map[string]any{
		"api_unificada": map[string]any{
			"custodias": []any{
				map[string]any{"operacaoId": "1900000000"},
			},
		},
	})

	_, err := ComputeHandler{}.Handle(context.Background(), msg, map[string]any{
		"target": "contrato_legado",
		"value": map[string]any{
			"field":              "api_unificada.custodias.0.operacaoId",
			"less_than_or_equal": 1900000000,
		},
	})
	if err != nil {
		t.Fatalf("compute handle: %v", err)
	}
	value, _ := msg.Get("contrato_legado")
	if value != true {
		t.Fatalf("contrato_legado = %v", value)
	}
}

func TestAssertAllPassesWithIndexedPaths(t *testing.T) {
	msg := slip.NewMessage("MSG", map[string]any{
		"api_unificada": map[string]any{
			"custodias": []any{
				map[string]any{
					"situacaoOperacao": "ATIVA",
					"convenio":         map[string]any{"codigoConvenio": "133341"},
				},
			},
		},
	})

	_, err := AssertHandler{}.Handle(context.Background(), msg, map[string]any{
		"all": []any{
			map[string]any{
				"field":  "api_unificada.custodias.0.convenio.codigoConvenio",
				"equals": "133341",
			},
			map[string]any{
				"field":  "api_unificada.custodias.0.situacaoOperacao",
				"equals": "ATIVA",
			},
		},
		"message": "Operacao fora dos criterios de convenio ou status.",
	})
	if err != nil {
		t.Fatalf("assert handle: %v", err)
	}
	value, _ := msg.Get("assert_passed")
	if value != true {
		t.Fatalf("assert_passed = %v", value)
	}
}

func TestAssertAllFailsWithMessage(t *testing.T) {
	msg := slip.NewMessage("MSG", map[string]any{"status": "CANCELADA"})

	_, err := AssertHandler{}.Handle(context.Background(), msg, map[string]any{
		"all": []any{
			map[string]any{"field": "status", "equals": "ATIVA"},
		},
		"message": "Operacao fora dos criterios",
	})
	if err == nil {
		t.Fatal("expected assert to fail")
	}
}

func TestJumpIfRedirectsWhenMatched(t *testing.T) {
	msg := slip.NewMessage("MSG", map[string]any{"contrato_legado": true})
	msg.AttachSlip([]slip.StepDef{
		{Name: "jump_if", Params: map[string]any{"field": "contrato_legado", "equals": true, "to": "finalizar"}},
		{Name: "enrich"},
		{ID: "finalizar", Name: "audit"},
	})
	step := slip.StepDef{Name: "jump_if", Params: map[string]any{"field": "contrato_legado", "equals": true, "to": "finalizar"}}

	_, err := JumpIfHandler{}.Handle(context.Background(), msg, step.Params)
	if err != nil {
		t.Fatalf("jump_if handle: %v", err)
	}
	cursor, changed, err := JumpIfHandler{}.NextCursor(msg, step, 0)
	if err != nil {
		t.Fatalf("jump_if next cursor: %v", err)
	}
	if !changed || cursor != 2 {
		t.Fatalf("cursor = %d changed = %v", cursor, changed)
	}
}
