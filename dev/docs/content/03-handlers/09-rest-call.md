---
sidebar_position: 12
sidebar_label: "Rest call"
---

# Rest call

Use `rest_call` quando o workflow precisa acionar uma API REST diretamente.

```yaml
- name: rest_call
  params:
    base_url: "https://mock.raysouz.studio"
    method: POST
    endpoint: /expedicao
    target: expedicao
```

Exemplo com POST, body e headers:

```yaml
- name: rest_call
  params:
    base_url: "https://mock.raysouz.studio"
    method: POST
    endpoint: /entregas
    target: entrega
    headers:
      x-correlation-id: "{correlation_id}"
    body:
      pedido_id: "{pedido_id}"
      itens: "{itens}"
    result_path: data
    timeout_ms: 3000
    required: true
```

Assim como no GraphQL, `required: false` permite continuar e marca resposta parcial.
