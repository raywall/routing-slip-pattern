---
sidebar_position: 7
sidebar_label: "State store"
---

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
  processing_lock:
    enabled: true
    ttl_seconds: 300
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

## Lock de processamento

Em execuções com muitas instancias, duas mensagens iguais podem chegar ao mesmo tempo antes de existir um snapshot salvo. Para evitar processamento concorrente do mesmo item, o runtime usa `processing_lock` por `message_id`.

```yaml
state_store:
  processing_lock:
    enabled: true
    ttl_seconds: 300
```

Com o lock ativo, apenas uma instancia processa o `message_id` por vez. Se outra instancia receber o mesmo item enquanto o primeiro processamento ainda está em andamento, ela não executa as etapas. No REST síncrono o runtime responde conflito; em filas, o evento permanece disponível para nova tentativa conforme a política do broker.

Se o `message_id` já estiver concluído, o runtime retorna o snapshot salvo sem reexecutar os steps. Isso protege integrações externas contra retries tardios e redeliveries de eventos.

Use um `message_id_path` estável e funcional. Ele deve representar o item processado, não um timestamp aleatório. Exemplos comuns são `event_id`, `order_id`, `document_id` ou outro identificador único do domínio.
