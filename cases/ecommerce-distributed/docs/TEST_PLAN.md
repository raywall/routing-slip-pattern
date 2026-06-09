# Plano de testes do case ecommerce-distributed

# Plano de testes do case ecommerce-distributed

Este plano possui quatro suites executáveis:

| Suite | Comando |
|---|---|
| Regressivos | `make ecommerce-regression` |
| Performance | `make ecommerce-performance COUNT=25 CONCURRENCY=4` |
| Caos | `make ecommerce-chaos` |
| MCP Server | `make ecommerce-mcp-test` |
| Todas | `make ecommerce-test-suite COUNT=25 CONCURRENCY=4` |

Cada execução grava evidências em `cases/ecommerce-distributed/results/` no formato JSON. Os arquivos `latest-*.json` mantêm o resultado mais recente por suite.

## Testes regressivos

| Teste | Evidencia esperada |
|---|---|
| `graphql_context` | GraphQL retorna `order`, `customer`, `inventory` e `deliveryPolicy` sem erros. |
| `workflow_happy_path` | Workflow REST conclui com HTTP 200 e sem campo `error`. |
| `metrics_available` | Metrics API responde uma lista de métricas. |
| `mcp_health` | MCP server responde health check. |

Comando:

```bash
make ecommerce-regression
```

Resultado:

- `results/latest-regression.json`
- status geral em `passed`;
- lista de checks com status HTTP, duração e identificadores relevantes.

## Testes de performance

| Teste | Como executar | Evidencia esperada |
|---|---|---|
| Carga REST concorrente | `make ecommerce-performance COUNT=100 CONCURRENCY=8` | Total, concluídos, falhos, tempo médio, p95, p99 e throughput. |
| Geração NDJSON | `make ecommerce-events-file COUNT=1000` | Arquivo de eventos para publicação posterior em Kafka/SQS. |
| Carga REST simples | `make ecommerce-load COUNT=25` | Disparo sequencial para comparação com a carga concorrente. |

Cada evento gerado recebe `event_id` único e `correlation_id` UUID v4 novo. Isso evita colisão com snapshots antigos no state store e permite medir execuções independentes.

Resultado:

- `results/latest-performance.json`;
- `summary.total`;
- `summary.completed`;
- `summary.failed`;
- `summary.average_ms`;
- `summary.p95_ms`;
- `summary.p99_ms`;
- `summary.throughput_per_second`.

## Testes de caos

| Falha | Configuracao | Resultado esperado |
|---|---|---|
| Payload inválido | Remove `order_id` do evento | Workflow retorna erro controlado e snapshot. |
| GraphQL indisponível | Para temporariamente `go-graphql-connector` | Workflow falha em `graphql_enrich` e preserva cursor. |
| Recuperação da integração | Religa `go-graphql-connector` e reenvia o mesmo evento | Workflow reprocessa do snapshot e conclui. |

Comando:

```bash
make ecommerce-chaos
```

Resultado:

- `results/latest-chaos.json`;
- evidência de erro controlado;
- evidência de reprocessamento com o mesmo `message_id`.

## Testes do MCP Server

| Teste | Tool | Evidencia esperada |
|---|---|---|
| Listar tools | `tools/list` | Lista contém `validate_workflow`, `explain_workflow`, `suggest_metrics`, `get_execution` e `list_state_snapshots`. |
| Validar workflow | `validate_workflow` | Retorno `valid: true` para o workflow carregado no runtime. |
| Explicar workflow | `explain_workflow` | Etapas compostas aparecem expandidas com handlers e integrações. |
| Sugerir métricas | `suggest_metrics` | Retorno com métricas e pontos de auditoria sugeridos. |
| Consultar execução | `get_execution` | Após executar um workflow, recupera snapshot por `message_id`. |

Comando:

```bash
make ecommerce-mcp-test
```

Resultado:

- `results/latest-mcp.json`;
- lista de tools encontradas;
- quantidade de steps explicados;
- snapshot recuperado do state store.

## Relatorio esperado

Registre os resultados em `results/` com:

- data e hora do teste;
- comando usado;
- quantidade de eventos;
- tempo total;
- p95/p99 quando disponivel;
- erros observados;
- `trace_id` de exemplos relevantes;
- observacoes de reprocessamento.
