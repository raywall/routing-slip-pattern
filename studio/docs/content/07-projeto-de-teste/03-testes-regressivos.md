# Testes regressivos

Os testes regressivos garantem que mudanças no runtime, nos handlers, no conector GraphQL, no Studio ou nos mocks não quebrem o comportamento esperado do cenário.

## Cenários

| Cenário | Objetivo | Entrada | Resultado esperado | Status atual |
|---|---|---|---|---|
| `happy-path` | Validar fluxo completo. | `payloads/happy-path.json` | Workflow conclui com status pronto para envio. | Preparado |
| `partial-data` | Validar dados incompletos. | `payloads/partial-data.json` | Fluxo para em validação funcional de estoque. | Preparado |
| `stop-and-reprocess` | Validar retomada por cursor. | `payloads/stop-and-reprocess.json` | Falha salva snapshot e reprocessamento continua do ponto correto. | Preparado |
| GraphQL context | Validar consulta de contexto. | Bruno `Consultar contexto ecommerce` | Retorna pedido, cliente, estoque e política. | Preparado |
| MCP validate | Validar contrato do workflow. | Bruno `Validar workflow ecommerce` | Retorno `valid: true`. | Preparado |
| MCP explain | Validar explicabilidade. | Bruno `Explicar workflow ecommerce` | Lista etapas, controles e integrações. | Preparado |
| MCP metrics | Validar sugestões de observabilidade. | Bruno `Sugerir metricas ecommerce` | Retorna métricas e pontos de auditoria. | Preparado |

## Comandos

```bash
make prepare
make run-ecommerce-case
make ecommerce-rest
```

Para gerar eventos sem enviar:

```bash
make ecommerce-events COUNT=100
```

Para enviar carga REST:

```bash
make ecommerce-load COUNT=25
```

## Resultados esperados

| Verificação | Como avaliar |
|---|---|
| Workflow expandido | O runtime carrega `workflow_ref` sem erro. |
| Correlação única | Cada evento gerado possui `correlation_id` UUID distinto. |
| Estado final | Payload possui `fulfillment_status: READY_FOR_SHIPMENT` no cenário feliz. |
| Auditoria | Logs/metricas registram recebido, contexto carregado, entrega planejada, operações concluídas e conclusão. |
| Reprocessamento | Cursor salvo aponta para a etapa que falhou. |
| Idempotência | Etapas já concluídas não precisam repetir efeito externo quando o state store está ativo. |

## Resultado registrado

Até o momento, a validação automatizada cobre:

- carga do workflow composto no teste Go do runtime;
- validação sintática dos YAMLs;
- validação JSON dos payloads, mocks e configs;
- validação do exemplo GraphQL `ecommerce-distributed`;
- validação do manifesto de documentação do Studio.

Os resultados de execução funcional contra mocks reais devem ser registrados em `cases/ecommerce-distributed/results/`.
