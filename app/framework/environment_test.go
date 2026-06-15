package framework

import "testing"

func TestExpandEnvironmentPreservesGraphQLVariables(t *testing.T) {
	t.Setenv("GRAPHQL_ENDPOINT", "http://graphql:8090/graphql")

	input := `endpoint: ${GRAPHQL_ENDPOINT:-http://localhost:8090/graphql}
query: query ($codigoCliente: String!) { customer(id: $codigoCliente) { id } }
missing: ${MISSING_VALUE:-fallback}`
	want := `endpoint: http://graphql:8090/graphql
query: query ($codigoCliente: String!) { customer(id: $codigoCliente) { id } }
missing: fallback`

	if got := expandEnvironment(input); got != want {
		t.Fatalf("expanded value:\n%s\nwant:\n%s", got, want)
	}
}

func TestExpandEnvironmentUsesEmptyValueWithoutDefault(t *testing.T) {
	if got := expandEnvironment("value: ${MISSING_VALUE}"); got != "value: " {
		t.Fatalf("expanded value = %q", got)
	}
}
