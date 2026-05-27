# Resiliencia por step

A resiliência por step permite que um workflow trate falhas transitórias sem espalhar retry, fallback e desvio de fluxo dentro dos handlers.

Ela é configurada diretamente no YAML:

```yaml
steps:
  - id: carregar-contexto
    name: graphql_enrich
    params:
      target: contexto
      required: true
    resilience:
      retry:
        attempts: 3
        backoff: exponential
        initial_interval_ms: 150
        max_interval_ms: 1000
        jitter: true
      on_failure:
        action: stop
```

## Retry

| Campo | O que faz |
|---|---|
| `attempts` | Total de tentativas da etapa. |
| `backoff` | Estratégia entre tentativas: `exponential`, `fixed` ou `none`. |
| `initial_interval_ms` | Espera inicial antes da próxima tentativa. |
| `max_interval_ms` | Limite máximo de espera. |
| `jitter` | Adiciona variação para evitar muitas chamadas ao mesmo tempo. |

## Ação após falha

| Ação | Comportamento |
|---|---|
| `stop` | Para o workflow e mantém o cursor na etapa atual. |
| `continue` | Registra a falha e continua para a próxima etapa. |
| `skip` | Marca a etapa como pulada e continua. |
| `jump` | Salta para um step específico usando `on_failure.to`. |

## Fallback

```yaml
steps:
  - id: consultar-servico
    name: rest_call
    params:
      base_url: https://api.example.test
      endpoint: /catalog/{sku}
      target: catalogo
      required: true
    resilience:
      retry:
        attempts: 2
        backoff: fixed
        initial_interval_ms: 200
      on_failure:
        action: jump
        to: fallback-catalogo

  - id: fallback-catalogo
    name: enrich
    params:
      data:
        catalogo_indisponivel: true
```

## Observabilidade

O histórico do step registra a tentativa final usada em `Attempt`.

As métricas também carregam:

- `attempt`;
- `trace_id`;
- `span_id`;
- `workflow`;
- `step`;
- `handler`;
- `failure_reason`, quando houver erro.

Isso permite identificar etapas instáveis, integrações que precisam de circuit breaker e fluxos que estão caindo em fallback com frequência.
