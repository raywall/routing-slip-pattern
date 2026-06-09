---
sidebar_position: 3
sidebar_label: "Paths e arrays"
---

# Paths e arrays

Paths sao caminhos em notação de ponto usados para ler valores dentro do payload. Eles aparecem em validações, asserts, computações, integrações, auditoria e filtros.

```json
{
  "order": {
    "id": "ORD-1001",
    "customer": {
      "id": "CUS-42"
    },
    "items": [
      { "sku": "SKU-1", "quantity": 2 },
      { "sku": "SKU-2", "quantity": 1 }
    ]
  }
}
```

Exemplos:

| Path | Valor |
| --- | --- |
| `order.id` | `ORD-1001` |
| `order.customer.id` | `CUS-42` |
| `order.items.0.sku` | `SKU-1` |
| `order.items.1.quantity` | `1` |

## Onde usar paths

Validação:

```yaml
- name: validate
  params:
    required:
      - order.id
      - order.customer.id
      - order.items.0.sku
```

Variáveis GraphQL:

```yaml
- name: graphql_enrich
  params:
    variables:
      orderID: "{order.id}"
```

Auditoria:

```yaml
- name: audit
  params:
    event: order.loaded
    fields:
      - correlation_id
      - order.id
      - order.customer.id
```

## Arrays

Para acessar uma posição especifica, use índice numérico. Para trabalhar com todos os itens, use `filter_array` ou uma expressão CEL.

```yaml
- name: filter_array
  params:
    source: order.items
    target: valid_items
    where:
      field: item.quantity
      greater_than: 0
```

Use arrays quando o workflow precisa limpar, validar ou selecionar subconjuntos antes de chamar uma integração ou decidir a proxima rota.

