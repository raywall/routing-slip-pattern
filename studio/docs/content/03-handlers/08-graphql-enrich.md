# GraphQL enrich

Use `graphql_enrich` para buscar dados no go-graphql-connector.

```yaml
- name: graphql_enrich
  params:
    query: "query ($pedidoID: String!) { dataSources(pedidoID: $pedidoID) { order { pedido_id status } } }"
    variables:
      pedidoID: "{pedido_id}"
    target: pedido
    result_path: dataSources.order
    required: true
```

Com variaveis interpoladas:

```yaml
- name: graphql_enrich
  params:
    endpoint: "${GRAPHQL_ENDPOINT:-http://localhost:8090/graphql}"
    query: "query ($sku: String!) { dataSources(sku: $sku) { catalogo { produtos { sku preco { valor } } } } }"
    variables:
      sku: "{itens.0.sku}"
    target: catalogo
    result_path: dataSources.catalogo
    timeout_ms: 3000
    required: true
```

Se `required: false`, falhas de endpoint marcam `<target>_partial: true` e permitem continuar.
