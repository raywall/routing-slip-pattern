---
sidebar_position: 4
sidebar_label: "Custom Business Metrics"
---

# Custom Business Metrics

O `custom-business-metrics` e a camada usada para observar o processamento em tempo real com métricas de negocio, eventos de etapa e dashboards configuráveis.

Ele complementa logs técnicos: em vez de responder apenas "a aplicação esta de pe?", ele ajuda a responder "onde esta cada processamento?", "qual etapa falhou?", "quanto falta?" e "qual fluxo esta demorando mais?".

![Visão geral da interface do CBM](docs/images/cbm-interface.jpg)

Componentes principais:

| Componente | Papel |
|---|---|
| `agent` | Recebe eventos JSON via UDP, agrupa e encaminha em lote. |
| `service` | API HTTP de ingestão, consulta, agregação, retenção e dashboards. |
| `webview` | Interface para visualizar e editar dashboards. |
| `storage` | Memoria para desenvolvimento ou DynamoDB/DynamoDB Local para persistência. |

```mermaid
flowchart LR
    Router[Routing Slip] -->|métricas HTTP| Service[Metrics Service]
    Router -. UDP JSON .-> Agent[Metrics Agent]
    Agent -->|batch HTTP| Service
    Service --> Store[(DynamoDB ou memoria)]
    Webview[Webview] -->|consultas| Service
```

Na rastreabilidade distribuída, cada evento pode trazer `trace_id` e `span_id`. Isso permite consultar toda a trilha técnica de uma execução, mesmo quando ela passa pelo workflow, pelo GraphQL connector e por APIs externas.

```bash
curl "http://localhost:8080/v1/metrics/events?trace_id=4bf92f3577b34da6a3ce929d0e0e4736"
curl "http://localhost:8080/v1/metrics/trace/4bf92f3577b34da6a3ce929d0e0e4736"
```
