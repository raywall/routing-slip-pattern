# State store

O state store guarda snapshots de execução. Ele e o componente que permite que um workflow continue do ponto correto apos erro, parada ou reinicio.

Um snapshot contem:

- `message_id`;
- `workflow`;
- `workflow_version`;
- `status`;
- `cursor`;
- `payload`;
- `history`;
- `errors`;
- `step_states`;
- `trace_id`;
- timestamps.

## Tipos suportados

| Tipo | Uso |
| --- | --- |
| `memory` | Testes e execuções descartáveis. |
| `file` | Desenvolvimento local com snapshots JSON. |
| `dynamodb` | Ambientes distribuídos e execução local com container. |

## Exemplo com arquivo

```yaml
features:
  persistent_state_enabled: true

state_store:
  type: file
  path: .routing-slip-state
  idempotency:
    enabled: true
    key_template: "{workflow}:{message_id}:{step_index}:{step}"
```

## Exemplo com DynamoDB

```yaml
state_store:
  type: dynamodb
  table: routing-slip-state
  endpoint: http://dynamodb:8000
  region: us-east-1
  ttl_days: 30
```

A tabela usa chave composta:

| Campo | Tipo | Uso |
| --- | --- | --- |
| `pk` | String | `message_id`. |
| `sk` | String | valor fixo `state`. |

## Idempotência

Quando `idempotency.enabled` esta ativo, o runtime calcula uma chave por etapa. Se a etapa ja foi concluída com sucesso e o cursor voltar para ela, o runtime registra `idempotent_skip` e segue sem repetir o efeito.

Isso e essencial para etapas que chamam APIs, notificam usuários ou atualizam sistemas externos.

