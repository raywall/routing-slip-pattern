package slip

import "testing"

func TestMessageGetPathSupportsArrayIndex(t *testing.T) {
	msg := NewMessage("MSG", map[string]any{
		"api_unificada": map[string]any{
			"custodias": []any{
				map[string]any{
					"convenio": map[string]any{"codigoConvenio": "133341"},
				},
			},
		},
	})

	value, ok := msg.GetPath("api_unificada.custodias.0.convenio.codigoConvenio")
	if !ok {
		t.Fatal("expected indexed path to be found")
	}
	if value != "133341" {
		t.Fatalf("value = %v", value)
	}
}
