# Testes de caos

Os testes de caos validam se o workflow continua explicável e recuperável quando dependências falham, ficam lentas ou retornam dados inesperados.

## Hipóteses

| Hipótese | Expectativa |
|---|---|
| Uma API obrigatória falha | Workflow para, salva snapshot e permite reprocessamento. |
| Uma API opcional falha | Workflow continua quando a política declara `continue`. |
| Uma API fica lenta | Timeout e retry são aplicados. |
| Falhas repetidas ocorrem | Circuit breaker evita insistência excessiva. |
| Runtime é interrompido | State store permite retomar do cursor salvo. |
| Métricas ficam indisponíveis | O processamento não deve depender da visualização. |

## Cenários

| Cenário | Como simular | Resultado esperado |
|---|---|---|
| `slow-connector` | Mock com atraso acima do timeout. | Retry, timeout e registro da falha. |
| `retry-success` | Mock falha nas primeiras chamadas e depois responde. | Workflow conclui após retry. |
| `circuit-open` | Mock retorna 503 repetidamente. | Circuit breaker abre no connector. |
| Falha de notificação | Endpoint de notificação retorna 500. | Workflow continua, pois a etapa é opcional. |
| Interrupção do runtime | Parar container no meio da execução. | Snapshot preserva cursor e payload. |
| Reenvio duplicado | Enviar mesmo `event_id`. | Idempotência evita repetir etapas concluídas. |

## Evidências

Durante os testes, registre:

- `event_id`;
- `correlation_id`;
- `trace_id`;
- etapa que falhou;
- política de resiliência aplicada;
- quantidade de tentativas;
- cursor salvo;
- resultado do reprocessamento;
- diferença de tempo entre execução original e reprocessamento.

## Resultados esperados

| Resultado | Critério |
|---|---|
| Falha visível | A etapa, o handler e o motivo aparecem nos logs/metricas. |
| Snapshot correto | O cursor aponta para o ponto de retomada. |
| Reprocessamento seguro | Etapas concluídas não precisam repetir side effects. |
| Trace preservado | A execução pode ser investigada por `trace_id`. |
| MCP útil | `get_execution`, `explain_workflow` e `suggest_metrics` ajudam a investigar. |

## Resultado registrado

Os cenários e mocks estão preparados. A execução real dos testes de caos deve ser feita com os mocks configurados no `mock.raysouz.studio` ou em um ambiente local equivalente, registrando os resultados em `cases/ecommerce-distributed/results/`.
