# GraphQL enrich

Use `graphql_enrich` para buscar dados no go-graphql-connector.

## Sintaxe

| Parametro | Obrigatorio | Descricao |
|---|---:|---|
| `endpoint` | Nao | URL do GraphQL Connector. Se omitido, usa configuracao do runtime. |
| `query` | Sim | Query GraphQL. Pode usar variaveis GraphQL. |
| `variables` | Nao | Mapa de variaveis, com interpolacao por `{path.do.payload}`. |
| `target` | Sim | Campo onde o resultado sera gravado no payload. |
| `result_path` | Nao | Caminho dentro da resposta GraphQL a ser extraido. |
| `timeout_ms` | Nao | Timeout da chamada. |
| `required` | Nao | Se `true`, falhas interrompem o workflow. Se `false`, marca parcialidade e segue. |

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

## Extraindo somente o trecho util

Uma resposta GraphQL normalmente vem no formato:

```json
{
  "dataSources": {
    "customer": {
      "id": "cust-42",
      "status": "ACTIVE"
    }
  }
}
```

Use `result_path` para salvar apenas o campo relevante:

```yaml
- name: graphql_enrich
  params:
    endpoint: "${GRAPHQL_ENDPOINT:-http://localhost:8090/graphql}"
    query: "query ($customerID: String!) { dataSources(customerID: $customerID) { customer { id status riskSegment creditLimit sourceSystem } } }"
    variables:
      customerID: "{customer_id}"
    target: customer
    result_path: dataSources.customer
    timeout_ms: 3000
    required: true
```

Depois da etapa, o payload passa a ter:

```json
{
  "customer": {
    "id": "cust-42",
    "status": "ACTIVE",
    "riskSegment": "LOW",
    "creditLimit": 2500,
    "sourceSystem": "api-mock-service"
  }
}
```

## Boas praticas

- Prefira variaveis GraphQL em vez de concatenar valores dentro da query.
- Use `required: true` para dados obrigatorios ao processo.
- Use `required: false` para enriquecimentos complementares.
- Combine `graphql_enrich` com `assert`, `condition`, `filter_array` ou `cel` para decidir os proximos passos.
