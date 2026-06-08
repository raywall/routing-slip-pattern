---
name: custom-business-metrics-observability
description: Use when instrumenting applications or routing-slip workflows with custom-business-metrics, designing metric events, dashboards, widget queries, filters, webview layouts, DynamoDB storage, trace/correlation lookup, or local service/agent/webview setup.
---

# Custom Business Metrics Observability

Use this skill to design business observability with `custom-business-metrics`.

## Components

- `service`: HTTP API for ingestion, query, aggregation, retention and dashboards.
- `agent`: UDP receiver on `8125`, forwards batches to service.
- `webview`: static dashboard UI, usually on `5173`.
- DynamoDB: optional persistent storage with TTL and indexes.

Common URLs:

```text
Metrics API:     http://localhost:8080
Metrics Webview: http://localhost:5173
UDP Agent:       :8125
```

## Bootstrap Guidance

Current modules are split by component:

```text
custom-business-metrics/service
custom-business-metrics/agent
custom-business-metrics/testapp
```

When the project becomes available through `pkg.go.dev`, generate application code with this intended flow:

1. Load service configuration from env/file.
2. Choose storage: memory for local demos, DynamoDB for persistence.
3. Start service HTTP API.
4. Optionally start UDP agent forwarding to `POST /v1/metrics`.
5. Serve webview static assets or deploy them behind a web server/CDN.

Pseudo-code:

```go
func main() {
    cfg := metrics.LoadConfig()
    store := metrics.NewStore(cfg.Storage)
    service := metrics.NewService(store, cfg)
    agent := metrics.NewAgent(cfg.Agent, service.IngestURL())

    go agent.Run(context.Background())
    service.Run(context.Background())
}
```

Before writing concrete Go constructors, inspect the installed package API. If the public API is not available yet, generate component commands/env vars instead of invented symbols.

Environment patterns:

```text
STORAGE_BACKEND=memory
STORAGE_BACKEND=dynamodb
DYNAMODB_TABLE=custom-business-metrics-events
DYNAMODB_ENDPOINT=http://localhost:4566
AWS_REGION=us-east-1
AGENT_UDP_ADDR=:8125
SERVICE_INGEST_URL=http://localhost:8080/v1/metrics
```

## Metric Event Contract

```json
{
  "name": "routing_slip.step.completed",
  "kind": "count",
  "value": 1,
  "unit": "event",
  "workflow": "order-processing",
  "step": "graphql_enrich",
  "status": "success",
  "source": "routing-slip-pattern",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "tags": {
    "message_id": "MSG-001",
    "correlation_id": "corr-001",
    "handler": "graphql_enrich",
    "duration_ms": "142",
    "order_id": "ORD-1001"
  },
  "timestamp": "2026-05-26T12:00:00Z"
}
```

Rules:

- Always include `workflow`, `step`, `status`, `source`, `correlation_id` and `trace_id` when available.
- Use `correlation_id` for business process lookup.
- Use `trace_id` for distributed technical tracing across routing-slip, GraphQL and external APIs.
- Put domain-specific attributes in `tags`.
- Redact secrets: authorization, tokens, passwords, api keys, client secrets.

## Ingestion

HTTP batch:

```bash
curl -X POST http://localhost:8080/v1/metrics \
  -H 'content-type: application/json' \
  -d '{"events":[{...}]}'
```

UDP agent receives newline-delimited JSON metric events and forwards to:

```text
SERVICE_INGEST_URL=http://localhost:8080/v1/metrics
AGENT_UDP_ADDR=:8125
```

## Query Endpoints

- `POST /v1/metrics`: ingest events
- `GET /v1/metrics/events`: raw events with filters
- `GET /v1/metrics/trace/{trace_id}`: events by trace
- `GET /v1/metrics`: summaries
- `GET /v1/metrics/series`: time series
- `GET /v1/dimensions`: known dimensions/tags
- `GET /v1/dashboards`: list dashboards
- `POST /v1/dashboards`: create/update dashboards

Examples:

```bash
curl "http://localhost:8080/v1/metrics/events?correlation_id=corr-001"
curl "http://localhost:8080/v1/metrics/trace/4bf92f3577b34da6a3ce929d0e0e4736"
curl "http://localhost:8080/v1/metrics/events?workflow=order-processing&status=failed"
```

## Webview

The webview supports:

- API URL/token configuration through settings.
- period picker.
- process list with filters using `key:value`.
- process detail popup with stage timeline.
- editable widget dashboard above process list.
- localStorage persistence for dashboard layout.

Filter examples:

```text
correlation_id:corr-001
trace_id:4bf92f3577b34da6a3ce929d0e0e4736
status:failed
order_id:ORD-1001
handler:graphql_enrich
```

## Widget Query Syntax

```text
aggregation:metric{filters}.rollup(function, seconds)
```

Examples:

```text
count:processes{*}
count:processes{status:completed}
count:processes{status:failed}
count:reprocesses{*}
avg:duration_ms{status:completed}
p95:duration_ms{*}
top:workflow{status:failed}
table:processes{status:failed}
count:processes{group_by:status}
count:processes{*}.rollup(count, 60)
```

Widget types:

- `bar chart`
- `pie chart`
- `point plot`
- `query value`
- `table`
- `timeseries`
- `top list`

## DynamoDB Storage

When `STORAGE_BACKEND=dynamodb`, configure:

```text
DYNAMODB_TABLE=custom-business-metrics-events
DYNAMODB_ENDPOINT=http://localhost:4566
AWS_REGION=us-east-1
```

Expected indexes:

- `correlation-index`
- `trace-index`

Use TTL via `expires_at` for retention.

## Dashboard Design Checklist

For workflow observability dashboards, include:

1. total processes in period
2. success/failure/reprocess counts
3. bar chart per hour for last 24h
4. p95 and average duration
5. failure ranking by workflow/step/handler
6. process list with `key:value` search
7. detail popup showing input, rule, output, status and failure reason per step
8. trace and correlation links

## Integration With routing-slip-pattern

Routing Slip emits step metrics through middleware. When authoring workflows, add semantic tags through `audit`, `log`, `datadog_metric`, and stable step ids. The metrics service should receive:

- `routing_slip.step.started`
- `routing_slip.step.completed`
- `routing_slip.step.failed`
- `routing_slip.step.duration_ms`

Prefer one `correlation_id` across composed workflows.
