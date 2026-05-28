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
| Payload inválido | Runner remove `order_id`. | Workflow para no `validate` com erro controlado. |
| GraphQL indisponível | Runner para temporariamente `go-graphql-connector`. | Workflow falha em `graphql_enrich` e salva snapshot. |
| Recuperação | Runner religa GraphQL e reenvia o mesmo evento. | Workflow reprocessa do cursor salvo e conclui. |

Comando:

```bash
make ecommerce-chaos
```

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

O resultado mais recente fica em:

```text
cases/ecommerce-distributed/results/latest-chaos.json
```

O runner sempre tenta religar o `go-graphql-connector` antes de encerrar a suite.
