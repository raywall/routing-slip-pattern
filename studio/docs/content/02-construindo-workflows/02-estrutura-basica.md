---
sidebar_position: 2
sidebar_label: "Estrutura básica"
---

# Estrutura básica de um workflow

Todo workflow possui metadados e uma lista de etapas. O exemplo abaixo valida um evento, adiciona metadados, chama uma API e registra auditoria.

```yaml
name: order-processing
description: Processa evento de pedido recebido.
version: "1.0"
error_policy: stop
message_id_path: order_id
correlation_id_path: correlation_id

steps:
  - id: validate-input
    name: validate
    params:
      required:
        - order_id
        - customer_id

  - id: add-context
    name: enrich
    params:
      data:
        source: ONLINE_STORE
        priority: NORMAL

  - id: load-order
    name: rest_call
    params:
      base_url: https://api.example.test
      endpoint: /orders/{order_id}
      method: GET
      target: order
      required: true

  - id: audit-completed
    name: audit
    params:
      event: order.processing.completed
      fields:
        - correlation_id
        - order_id
        - order.status
```

## Campos do cabeçalho

| Campo | Obrigatório | Descrição |
| --- | --- | --- |
| `name` | Sim | Nome técnico do workflow. |
| `description` | Nao | Explica a finalidade do fluxo. |
| `version` | Nao | Versão funcional do workflow. |
| `error_policy` | Nao | `stop`, `continue` ou `skip`. |
| `message_id_path` | Recomendado | Path usado para identificar a execução e reprocessar. |
| `correlation_id_path` | Recomendado | Path usado para correlacionar logs, métricas e traces. |
| `steps` | Sim | Lista ordenada de etapas. |

Quando o payload nao traz o campo configurado em `correlation_id_path`, o runtime gera um UUID v4 automaticamente e o injeta antes da primeira etapa. Isso evita reutilizar identificadores fixos em testes e mantém logs, métricas e traces ligados ao mesmo processamento.

Use `message_id_path` para a chave funcional de retomada e idempotência. O `correlation_id` acompanha a jornada; o `message_id` aponta para o snapshot que permite continuar de onde parou.

## Estrutura de um step

```yaml
- id: nome-estavel-da-etapa
  name: handler_registrado
  params:
    chave: valor
  resilience:
    retry:
      attempts: 3
```

Use `id` sempre que a etapa puder ser alvo de salto, auditoria, explicação ou idempotência. O `name` e o handler executado. O `params` muda conforme o handler.
