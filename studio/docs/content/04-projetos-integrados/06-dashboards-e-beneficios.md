# Dashboards e beneficios

O dashboard permite acompanhar processos em tempo real sem abrir logs da aplicacao.

Indicadores uteis para routing slip:

| Indicador | O que mostra |
|---|---|
| Processamentos iniciados | Volume de entradas no periodo. |
| Processamentos concluidos | Quantidade finalizada com sucesso. |
| Falhas por etapa | Onde o processo mais quebra. |
| Tempo medio por step | Gargalos do workflow. |
| Integracoes externas | Volume de enriquecimentos e chamadas. |
| Timeline por correlation_id | Jornada detalhada de uma mensagem. |

Exemplo de consulta de widget:

```text
sum:routing_slip.step.completed{source:routing-slip-app}.as_count()
```

Com agrupamento:

```text
sum:routing_slip.step.failed{workflow:pedido-fulfillment} by {step}.as_count()
```

Beneficios:

- observabilidade granular por workflow, step e correlation id;
- transparencia para entender entrada, regra, saida e falha;
- reprocessamento mais seguro, porque o cursor e o historico ficam evidentes;
- menor impacto no fluxo principal quando eventos sao enviados via agent;
- dashboards parametrizaveis por JSON;
- retencao controlada por TTL quando usando DynamoDB.
