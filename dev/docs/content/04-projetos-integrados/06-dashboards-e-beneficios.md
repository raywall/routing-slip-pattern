# Dashboards e benefícios

O dashboard permite acompanhar processos em tempo real sem abrir logs da aplicação.

Indicadores uteis para routing slip:

| Indicador | O que mostra |
|---|---|
| Processamentos iniciados | Volume de entradas no período. |
| Processamentos concluídos | Quantidade finalizada com sucesso. |
| Falhas por etapa | Onde o processo mais quebra. |
| Tempo médio por step | Gargalos do workflow. |
| Integrações externas | Volume de enriquecimentos e chamadas. |
| Timeline por correlation_id | Jornada detalhada de uma mensagem. |

Exemplo de consulta de widget:

```text
sum:routing_slip.step.completed{source:routing-slip-app}.as_count()
```

Com agrupamento:

```text
sum:routing_slip.step.failed{workflow:pedido-fulfillment} by {step}.as_count()
```

Benefícios:

- observabilidade granular por workflow, step e correlation id;
- transparência para entender entrada, regra, Saida e falha;
- reprocessamento mais seguro, porque o cursor e o histórico ficam evidentes;
- menor impacto no fluxo principal quando eventos sao enviados via agent;
- dashboards parametrizáveis por JSON;
- retenção controlada por TTL quando usando DynamoDB.
