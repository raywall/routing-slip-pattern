# Resiliencia e escalabilidade

Workflows reais dependem de rede, APIs, bancos, filas e servicos externos. Esses componentes podem falhar temporariamente. A resiliencia por etapa permite declarar como o runtime deve reagir sem duplicar logica em cada handler.

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
| `jitter` | Varia o intervalo para evitar rajadas simultaneas. |

## Tratamento de falha

```yaml
resilience:
  on_failure:
    action: jump
    to: manual-review
```

Acoes disponiveis:

| Acao | Resultado |
| --- | --- |
| `stop` | Para o workflow e salva cursor da falha. |
| `continue` | Registra falha e segue. |
| `skip` | Marca etapa como pulada e segue. |
| `jump` | Vai para outro step. |

## Escalabilidade

O runtime pode ser acionado por REST, Kafka ou SQS. Para escalar, rode multiplas instancias consumindo eventos. O state store e a idempotencia ajudam a reduzir risco operacional quando ha reprocessamentos, retries ou concorrencia.

Use filas ou topicos quando o volume for alto, quando houver picos ou quando o produtor nao deve esperar a conclusao do workflow.

