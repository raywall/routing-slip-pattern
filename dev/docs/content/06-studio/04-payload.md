---
sidebar_position: 4
sidebar_label: "Payload de entrada"
---

# Payload de entrada


A aba **Payload de Entrada** define o JSON usado na simulação local do workflow. Pense nela como o evento inicial que chegaria por REST, Kafka ou SQS em uma execução real.

Esse payload e importante porque todos os handlers leem ou escrevem dados a partir dele. Por exemplo:

- `validate` verifica se campos obrigatórios existem.
- `graphql_enrich` usa valores do payload para montar variáveis da query.
- `condition`, `assert`, `compute`, `cel` e `filter_array` tomam decisões com base nos campos atuais.
- `audit`, `notify` e `rest_call` podem interpolar valores usando `{caminho.do.campo}`.

![Payload de envio](docs/images/studio-payload.jpg)

Um payload simples pode começar assim:

```json
{
  "correlation_id": "839e2b76-0aa1-47dc-96b1-67f41b73c795",
  "event_name": "PEDIDO_APROVADO",
  "pedido_id": "PED-9988",
  "customer_id": "cust-42",
  "received_at": "2026-05-26T12:00:00Z"
}
```

No Studio, ao carregar um exemplo ou executar um payload sem `correlation_id`, um novo UUID é criado para a simulação. Se você estiver validando reprocessamento, mantenha o mesmo payload/snapshot para preservar a relação entre execução original e retomada.

Use nomes estáveis para os campos principais. Eles acabam aparecendo nos logs, no resumo da execução e, quando o workflow estiver integrado ao runtime, nas metrics de observabilidade.

Atalhos uteis no editor de payload:

| Atalho | Ação |
|---|---|
| `Tab` | Indena com dois espaços. |
| `Shift+Tab` | Remove intentadão. |

Antes de executar, o Studio tenta interpretar o conteúdo como JSON. Se houver erro de sintaxe, corrija o payload antes de analisar o comportamento do workflow; assim voce evita investigar uma falha que nao pertence ao processo em si.
