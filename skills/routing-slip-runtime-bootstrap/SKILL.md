---
name: routing-slip-runtime-bootstrap
description: Use when generating a Go application, config.yaml, Docker/local setup, Lambda entrypoint, or deployment bootstrap for routing-slip-pattern runtime with REST, Kafka, SQS, SNS, state store, idempotency, metrics, tracing, and MCP.
---

# Routing Slip Runtime Bootstrap

Use this skill to generate the base application and configuration for running `routing-slip-pattern`.

## Main Artifacts To Produce

For a new solution, usually generate:

- `config.yaml`
- one or more workflow YAML files
- `main.go` or Lambda handler bootstrap
- environment variables
- Docker/compose snippets when requested
- sample REST/Kafka/SQS/SNS test events

## Config Contract

```yaml
service:
  name: order-workflow
  run_id: local-dev

trigger:
  connector: rest
  mode: sync
  rest:
    addr: ":8088"
    path: /process

features:
  tracing_enabled: true
  persistent_state_enabled: true
  mcp_enabled: true

state_store:
  type: file
  path: .routing-slip-state
  idempotency:
    enabled: true
    key_template: "{workflow}:{message_id}:{step_index}:{step}"
  processing_lock:
    enabled: true
    ttl_seconds: 300

metrics:
  endpoint: http://localhost:8080/v1/metrics
  segment: local
  source: routing-slip-app
  tags:
    env: local

mcp:
  enabled: true
  bind: 127.0.0.1:9091
  mode: readonly
  auth:
    type: none
```

## Trigger Patterns

REST sync:

```yaml
trigger:
  connector: rest
  mode: sync
  rest:
    addr: ":8088"
    path: /process
```

REST async:

```yaml
trigger:
  connector: rest
  mode: async
  rest:
    addr: ":8088"
    path: /process
```

Kafka:

```yaml
trigger:
  connector: kafka
  mode: async
  kafka:
    brokers: ["localhost:9092"]
    topic: order-events
    group_id: routing-slip-pattern
```

SQS:

```yaml
trigger:
  connector: sqs
  mode: async
  sqs:
    endpoint: http://localhost:4566
    region: us-east-1
    queue_url: http://localhost:4566/000000000000/order-events
    wait_time_seconds: 10
    max_messages: 10
    visibility_timeout: 30
```

SNS via subscribed SQS queue:

```yaml
trigger:
  connector: sns
  mode: async
  sns:
    endpoint: http://localhost:4566
    region: us-east-1
    queue_url: http://localhost:4566/000000000000/order-events-sns
    wait_time_seconds: 10
    max_messages: 10
```

## Go Bootstrap Guidance

Current module path is:

```text
github.com/raywall/routing-slip-pattern
```

Future package availability in `pkg.go.dev` should make app bootstraps simpler. Until public APIs are finalized, generate code using the exported package surface available in the target version and include a note to check current package names.

For CLI/runtime apps, the intended flow is:

1. Load `config.yaml`.
2. Load workflow YAML.
3. Build router/runtime with all standard handlers.
4. Enable state store, idempotency, metrics, tracing and MCP from config.
5. Start the configured connector (`rest`, `kafka`, `sqs`, `sns`).

Pseudo-code:

```go
package main

func main() {
    cfg := mustLoadConfig("config.yaml")
    workflow := mustLoadWorkflow("workflow.yaml")
    runtime := routing.NewRuntime(cfg, workflow)
    runtime.MustRegisterDefaultHandlers()
    runtime.Run(context.Background())
}
```

When generating real code, inspect the installed package API and adapt the pseudo-code rather than inventing non-existent symbols.

## Lambda Bootstrap Guidance

For Lambda:

- Keep workflow/config in the deployment package or layer, commonly under `/opt/routing-slip`.
- Build a Go Lambda binary that imports the framework at compile time.
- Layer is operational content, not a Go module injection mechanism.
- Handler should parse event payload, create/load message, run the workflow and return status/result.

Pseudo-code:

```go
func handler(ctx context.Context, payload map[string]any) (map[string]any, error) {
    msg := routing.NewMessageFromPayload(payload)
    result, err := runtime.Process(ctx, msg)
    return result.AsMap(), err
}
```

## Required Runtime Qualities

Always configure:

- stable `message_id_path`
- `correlation_id_path`
- idempotency key template
- state store with resume support
- tracing enabled
- metrics endpoint when `custom-business-metrics` is available
- MCP readonly gateway for diagnostics when safe

## Output Checklist

When asked to bootstrap a service, return:

1. Suggested folder structure.
2. `config.yaml`.
3. minimal `main.go` or Lambda handler.
4. workflow path conventions.
5. local run commands.
6. test payload or event.
7. notes for AWS/LocalStack endpoints.
