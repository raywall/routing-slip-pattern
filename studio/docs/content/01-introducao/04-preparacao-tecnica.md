# Preparacao tecnica

A Fase 0 cria a base para evoluir o projeto por etapas sem obrigar o usuario a mudar todos os fluxos de uma vez.

Ela define tres coisas importantes:

- feature flags para ligar recursos novos quando fizer sentido;
- convencao comum de identificadores;
- politica de mascaramento de campos sensiveis.

## Feature flags

No `routing-slip-pattern`, as flags ficam no `config.yaml`:

```yaml
features:
  tracing_enabled: true
  mcp_enabled: false
  async_metrics_enabled: false
  persistent_state_enabled: false
```

| Flag | Uso |
|---|---|
| `tracing_enabled` | Liga a propagacao de `traceparent`, `trace_id` e spans. |
| `mcp_enabled` | Reserva para habilitar o MCP Gateway em fases futuras. |
| `async_metrics_enabled` | Reserva para emissao assincrona de metricas. |
| `persistent_state_enabled` | Reserva para state store persistente. |

## Convencao de identificadores

| Campo | Significado |
|---|---|
| `message_id` | Mensagem processada pelo runtime. |
| `correlation_id` | Processo de negocio. |
| `trace_id` | Trilha tecnica distribuida. |
| `span_id` | Etapa ou chamada dentro do trace. |
| `attempt` | Tentativa de processamento. |
| `workflow` | Nome do workflow. |
| `step` | Etapa do workflow. |
| `handler` | Handler que executa a etapa. |

Essa convencao e usada por logs, historico, metricas, dashboards e futuras ferramentas MCP.

## Mascaramento de dados sensiveis

Campos sensiveis nao devem aparecer em logs, metricas, respostas MCP ou diagnosticos.

```yaml
security:
  redaction:
    enabled: true
    fields:
      - authorization
      - client_secret
      - access_token
      - refresh_token
      - password
      - token
      - api_key
      - x-api-key
      - x-serial-number
```

O objetivo e permitir observabilidade profunda sem transformar a propria observabilidade em risco operacional.
