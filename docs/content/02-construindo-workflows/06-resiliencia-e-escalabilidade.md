---
sidebar_position: 6
sidebar_label: "Resiliência e escalabilidade"
---

# Resiliência e escalabilidade

Workflows reais dependem de rede, APIs, bancos, filas e serviços externos. Esses componentes podem falhar temporariamente. A resiliência por etapa permite declarar como o runtime deve reagir sem duplicar logica em cada handler.

## Retry com backoff

```yaml
- id: load-catalog
  name: graphql_enrich
  params:
    query: "query ($sku: String!) { dataSources(sku: $sku) { product { sku status } } }"
    variables:
      sku: "{items.0.sku}"
    target: product
    result_path: dataSources.product
    required: true
  resilience:
    retry:
      attempts: 3
      backoff: exponential
      initial_interval_ms: 200
      max_interval_ms: 1500
      jitter: true
    on_failure:
      action: stop
```

| Campo | Uso |
| --- | --- |
| `attempts` | Quantas tentativas no total. |
| `backoff` | `fixed`, `exponential` ou `none`. |
| `initial_interval_ms` | Intervalo inicial entre tentativas. |
| `max_interval_ms` | Limite de espera. |
| `jitter` | Varia o intervalo para evitar rajadas simultâneas. |

## Tratamento de falha

```yaml
resilience:
  on_failure:
    action: jump
    to: manual-review
```

Acoes disponíveis:

| Ação | Resultado |
| --- | --- |
| `stop` | Para o workflow e salva cursor da falha. |
| `continue` | Registra falha e segue. |
| `skip` | Marca etapa como pulada e segue. |
| `jump` | Vai para outro step. |

## Escalabilidade

O runtime pode ser acionado por REST, Kafka ou SQS. Para escalar, rode múltiplas instancias consumindo eventos. O state store e a idempotência ajudam a reduzir risco operacional quando ha reprocessamentos, retries ou concorrência.

Use filas ou tópicos quando o volume for alto, quando houver picos ou quando o produtor nao deve esperar a conclusão do workflow.

