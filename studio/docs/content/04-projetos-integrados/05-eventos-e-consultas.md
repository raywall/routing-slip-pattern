# Eventos e consultas

Um evento de metrica representa algo que aconteceu no processo.

```json
{
  "name": "routing_slip.step.completed",
  "kind": "count",
  "value": 1,
  "unit": "event",
  "workflow": "pedido-fulfillment",
  "step": "graphql_enrich",
  "status": "success",
  "source": "routing-slip-app",
  "tags": {
    "message_id": "MSG-001",
    "correlation_id": "corr-abc",
    "handler": "graphql_enrich",
    "duration_ms": "37"
  },
  "timestamp": "2026-05-23T12:00:00Z"
}
```

Eventos recomendados para routing slip:

- `routing_slip.workflow.started`;
- `routing_slip.workflow.completed`;
- `routing_slip.workflow.failed`;
- `routing_slip.step.started`;
- `routing_slip.step.completed`;
- `routing_slip.step.failed`;
- `routing_slip.step.skipped`;
- `routing_slip.payload.enriched`;
- `routing_slip.decision.evaluated`.

Consultas expostas pelo service:

| Endpoint | Uso |
|---|---|
| `POST /v1/metrics` | Ingestao de eventos. |
| `GET /v1/metrics/events` | Lista eventos crus filtrados. |
| `GET /v1/metrics` | Retorna agregacoes/sumarios. |
| `GET /v1/metrics/series` | Retorna serie temporal por bucket. |
| `GET /v1/metrics/dimensions` | Lista dimensoes/tags disponiveis. |
| `GET /v1/dashboards` | Lista dashboards. |
| `POST /v1/dashboards` | Salva dashboard. |
| `DELETE /v1/dashboards/{id}` | Remove dashboard. |
