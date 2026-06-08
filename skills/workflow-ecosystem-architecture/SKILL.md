---
name: workflow-ecosystem-architecture
description: Use when designing an end-to-end solution combining routing-slip-pattern, go-graphql-connector, and custom-business-metrics, including architecture, folder structure, local stack, responsibilities, integration boundaries, observability, test strategy, and generated artifacts.
---

# Workflow Ecosystem Architecture

Use this skill to design complete solutions with the three workflow projects.

## Project Responsibilities

| Project | Responsibility |
|---|---|
| `routing-slip-pattern` | Workflow runtime, YAML steps, triggers, state store, idempotency, reprocessing, MCP, Studio. |
| `go-graphql-connector` | GraphQL anticorruption/read facade over external APIs and data sources. |
| `custom-business-metrics` | Business/process observability, metrics ingestion, storage, dashboard, trace/correlation lookup. |

## Recommended Architecture

```text
Event/REST/Kafka/SQS/SNS
  -> routing-slip-pattern
      -> validate/assert/cel/condition
      -> graphql_enrich for read/enrichment
          -> go-graphql-connector
              -> REST/DynamoDB/S3/RDS/Redis/external APIs
      -> aws_action/rest_call for commands/effects
      -> audit/log/datadog_metric
      -> state store/idempotency/reprocess
      -> custom-business-metrics
          -> service/agent/webview/DynamoDB
```

## Solution Folder Pattern

```text
solution/
  routing/
    config.yaml
    workflows/
      service-a/main.yaml
      service-b/child.yaml
  graphql/
    config/
      service.json
      schema.json
      connectors.json
      mock.json
  metrics/
    dashboards/
      operations.json
  cmd/
    workflow-runtime/main.go
    graphql/main.go
  docker-compose.yml
  Makefile
```

## Design Rules

- Reads and enrichment go through `go-graphql-connector`.
- Commands/effects use `routing-slip-pattern` handlers such as `aws_action`, `rest_call`, `notify`.
- Business/process visibility goes to `custom-business-metrics`.
- Use one `correlation_id` for the whole business process, including composed workflows.
- Use `trace_id`/`traceparent` for technical propagation across routing-slip, GraphQL and APIs.
- Do not let child workflows generate a new correlation id.
- Use state store and idempotency for every workflow that may be reprocessed.
- Keep workflow files small with `workflow_ref`; export composed YAML only for deployment when required.

## Local Stack Ports

Common local ports:

```text
routing-slip REST:       8088
routing-slip Studio:     8089
routing-slip MCP:        9091
go-graphql-connector:    8090
metrics service:         8080
metrics webview:         5173
metrics UDP agent:       8125/udp
LocalStack:              4566
```

## Artifacts To Generate For Users

When building a new solution, produce:

1. routing-slip workflow YAML
2. routing-slip `config.yaml`
3. input payload examples
4. GraphQL `service.json`
5. GraphQL `schema.json`
6. GraphQL `connectors.json`
7. GraphQL query examples
8. metrics dashboard/widget suggestions
9. local compose/Makefile commands
10. bootstrap `main.go` files or pseudo-code tied to installed package APIs
11. test events for REST/Kafka/SQS/SNS
12. reprocessing/idempotency validation scenario

## Testing Strategy

Regression:

- valid payload completes
- missing required field fails at `validate`
- condition/assert/CEL branches behave as expected
- workflow_ref composition preserves correlation id
- GraphQL enrich stores expected target
- side effect handlers are idempotent or guarded

Performance:

- run parallel REST events
- run Kafka/SQS batches
- measure p50/p95 duration
- track GraphQL connector latency
- track metric ingestion rate

Chaos/resilience:

- GraphQL connector returns 500/timeout
- external REST returns 404/500
- circuit breaker opens
- state store unavailable
- process stops mid-flow and resumes from cursor

Observability:

- process visible by `correlation_id`
- trace visible by `trace_id`
- dashboard shows success/failure/reprocess
- step detail shows input/rule/output/status/failure

MCP:

- `list_handlers`
- `validate_workflow`
- `explain_workflow`
- `generate_workflow_from_business_rules`
- `validate_workflow_against_business_rules`
- `suggest_metrics`
- `get_execution`

## Model Behavior

When helping a user:

- Ask for domain/event/input fields only if not inferable.
- Prefer generating working starter artifacts over abstract explanation.
- Keep examples neutral and non-sensitive.
- Mention that future `pkg.go.dev` availability should simplify imports; inspect current package APIs before writing final Go code.
- Avoid inventing APIs. If public constructor names are unknown, provide pseudo-code plus TODO comments, or inspect local modules.
