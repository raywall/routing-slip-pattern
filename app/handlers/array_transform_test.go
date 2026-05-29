package handlers

import (
	"context"
	"testing"

	"github.com/raywall/routing-slip-pattern/slip"
)

func TestArrayTransformHandlerUpdatesAndFiltersNestedArrays(t *testing.T) {
	msg := slip.NewMessage("msg-1", map[string]any{
		"data": map[string]any{
			"codigo_identificacao_convenio": "133341",
		},
		"operacoes": []any{
			map[string]any{
				"codigoEmpresaContabil":  341,
				"codigoSituacaoOperacao": 1,
				"convenio":               map[string]any{"codigoConvenio": "133341"},
				"parcelas": []any{
					map[string]any{"dataVencimento": "2025-01-01", "valorParcelaOriginal": 150.0},
					map[string]any{"dataVencimento": "2024-12-31", "valorParcelaOriginal": 90.0},
				},
			},
			map[string]any{
				"codigoEmpresaContabil":  29,
				"codigoSituacaoOperacao": 2,
				"convenio":               map[string]any{"codigoConvenio": "999999"},
				"parcelas":               []any{},
			},
		},
	})

	_, err := ArrayTransformHandler{}.Handle(context.Background(), msg, map[string]any{
		"source":  "operacoes",
		"target":  "operacoes_filtradas",
		"filters": map[string]any{"expr": "item.convenio.codigoConvenio == data.codigo_identificacao_convenio && item.codigoSituacaoOperacao == 1"},
		"updates": []any{
			map[string]any{
				"when": map[string]any{"field": "item.codigoEmpresaContabil", "equals": 341},
				"set":  map[string]any{"codigoDependenciaEmpresaOperanteCredito": "05961"},
			},
		},
		"nested": []any{
			map[string]any{
				"source":  "parcelas",
				"filters": map[string]any{"expr": "item.dataVencimento > '2024-12-31'"},
			},
		},
	})
	if err != nil {
		t.Fatalf("array_transform handle: %v", err)
	}

	raw, ok := msg.GetPath("operacoes_filtradas")
	if !ok {
		t.Fatal("operacoes_filtradas not found")
	}
	items := raw.([]any)
	if got, want := len(items), 1; got != want {
		t.Fatalf("filtered operations = %d, want %d", got, want)
	}
	item := items[0].(map[string]any)
	if got, want := item["codigoDependenciaEmpresaOperanteCredito"], "05961"; got != want {
		t.Fatalf("credit department = %v, want %v", got, want)
	}
	parcelas := item["parcelas"].([]any)
	if got, want := len(parcelas), 1; got != want {
		t.Fatalf("filtered installments = %d, want %d", got, want)
	}
}
