# Plano de testes do case ecommerce-distributed

## Performance

| Teste | Como executar | Evidencia esperada |
|---|---|---|
| Carga REST simples | `make ecommerce-load COUNT=100` | Throughput, tempo medio e p95 no metrics service. |
| Geração NDJSON | `make ecommerce-events COUNT=1000` | Arquivo de eventos para publicacao posterior em Kafka/SQS. |
| GraphQL sob carga | Repetir consulta Bruno ou script externo | Tempo de resposta e retries por connector. |

## Resiliencia

| Falha | Configuracao | Resultado esperado |
|---|---|---|
| API lenta | Mock `slow-connector` | Timeout, retry e erro classificado. |
| Erro transitorio | Mock `retry-success` | Sucesso apos retry sem parar workflow. |
| API indisponivel | Mock `circuit-open` | Circuit breaker abre e o workflow salva snapshot. |
| Notificacao falha | Endpoint de notificacao retorna 500 | Workflow continua porque `required: false` e `on_failure: continue`. |
| Runtime interrompido | Parar container durante execucao | State store preserva cursor para reprocessamento. |

## Escalabilidade

| Teste | Objetivo |
|---|---|
| Aumentar volume Kafka | Avaliar throughput e lag. |
| Aumentar volume SQS | Avaliar drenagem gradual e reprocessamento. |
| Repetir mesmo `event_id` | Validar idempotencia por etapa. |
| Executar com métricas indisponíveis | Garantir que emissão de métricas nao bloqueia workflow. |

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
