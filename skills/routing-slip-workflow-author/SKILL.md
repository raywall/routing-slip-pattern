---
name: routing-slip-workflow-author
description: Use when creating, reviewing, refactoring, linting, composing, or explaining routing-slip-pattern workflow YAML, including handlers, business rules, workflow_ref composition, CEL, idempotency, observability, and payload examples.
---

# Routing Slip Workflow Author

Use this skill to help users design workflow YAML for `routing-slip-pattern`.

## Core Workflow Contract

Prefer this shape:

```yaml
name: order-fulfillment
description: Processa um evento de pedido confirmado.
version: draft
error_policy: stop
message_id_path: event_id
correlation_id_path: correlation_id

steps:
  - id: validate-input
    name: validate
    params:
      required:
        - event_id
        - correlation_id
        - order_id
```

Rules:

- Always add stable `id` for steps. Ids are used by logs, diagrams, `jump_if`, reprocessing, idempotency and human review.
- Prefer `message_id_path` as a stable event/process id. Prefer `correlation_id_path` as UUID-like e2e identifier.
- Use comments before important steps to preserve source/rationale. Do not remove user comments.
- Use examples that are domain-neutral, such as orders, catalog, inventory, delivery, notification or document processing.

## Handler Syntax

### validate

```yaml
- id: validate-input
  name: validate
  params:
    required:
      - correlation_id
      - order_id
```

### condition

Stops the flow functionally when the rule does not match.

```yaml
- id: only-approved
  name: condition
  params:
    field: order.status
    equals: APPROVED
```

### assert

Fails the workflow when a mandatory condition is false.

```yaml
- id: assert-order-ready
  name: assert
  params:
    all:
      - field: order.status
        equals: APPROVED
      - field: order.items
        min_items: 1
    message: Order is not ready.
```

Supported comparisons: `equals`, `not_equals`, `less_than`, `less_than_or_equal`, `greater_than`, `greater_than_or_equal`, `min_items`, `max_items`, `exists`.

### compute

```yaml
- id: compute-has-items
  name: compute
  params:
    target: has_items
    value:
      field: order.items
      min_items: 1
```

### cel

Use CEL for readable multi-field rules. `on_false` can be `error`, `fail`, `jump`, `continue` or `stop`.

```yaml
- id: validate-cel
  name: cel
  params:
    expr: "order.status == 'APPROVED' && size(order.items) > 0"
    target: rules.order_ready
    on_false: error
    message: Order is not valid.
```

### graphql_enrich

Use for read/enrichment through `go-graphql-connector`.

```yaml
- id: load-order-context
  name: graphql_enrich
  params:
    endpoint: http://localhost:8090/graphql
    query: "query ($orderID: String!) { dataSources(orderID: $orderID) { order { id status total } } }"
    variables:
      orderID: "{order_id}"
    target: order_context
    result_path: dataSources
    timeout_ms: 3000
    required: true
```

### rest_call

Use for direct REST effects or reads that should not go through GraphQL.

```yaml
- id: call-delivery-api
  name: rest_call
  params:
    base_url: https://api.example.test
    endpoint: /delivery/orders/{order_id}
    method: POST
    body:
      order_id: "{order_id}"
    target: delivery_result
    required: true
```

### aws_action

Use for controlled side effects in AWS/LocalStack.

```yaml
- id: publish-order-ready
  name: aws_action
  params:
    service: sqs
    action: send
    endpoint: http://localstack:4566
    region: us-east-1
    queue_url: http://localstack:4566/000000000000/order-events
    message:
      type: ORDER_READY
      order_id: "{order_id}"
    target: sqs_result
    required: true
```

Supported services/actions:

- `dynamodb`: `put`, `get`, `update`, `delete`
- `s3`: `put`, `get`, `delete`
- `sqs`: `send`
- `sns`: `publish`
- `secretsmanager`: `get`, `create`, `put`, `update`, `delete`
- `ssm`: `get`, `put`, `create`, `update`, `delete`

### filter_array

```yaml
- id: filter-valid-items
  name: filter_array
  params:
    source: order.items
    target: valid_items
    where:
      field: item.quantity
      greater_than: 0
```

### array_transform

```yaml
- id: enrich-items
  name: array_transform
  params:
    source: order.items
    target: enriched_items
    filters:
      expr: "item.quantity > 0"
    updates:
      - when:
          field: item.warehouse
          equals: MAIN
        set:
          priority: HIGH
```

### jump_if

Only jumps forward.

```yaml
- id: jump-legacy
  name: jump_if
  params:
    field: order.legacy
    equals: true
    to: audit-completed
```

### workflow_ref

Use workspace-root paths, not `../`, when composing workflows in Studio.

```yaml
- id: reserve-delivery
  name: workflow_ref
  params:
    file: delivery/reserve-and-delivery
    prefix: delivery
```

Referenced workflows inherit `message_id`, `correlation_id`, `trace_id`, headers and payload. They must not create a new correlation id.

### log, audit, datadog_metric, notify

```yaml
- id: log-ready
  name: log
  params:
    level: info
    message: "Order {order_id} ready"
    fields: [correlation_id, order_id]
    required: false

- id: metric-completed
  name: datadog_metric
  params:
    metric: routing_slip.orders.completed
    type: count
    value: 1
    tags:
      workflow: order-fulfillment
    required: false

- id: audit-completed
  name: audit
  params:
    event: order_fulfillment.completed
    fields: [correlation_id, order_id]
```

## Business Rules Metadata

When a project contains business rules, preserve this schema:

```yaml
technical_metadata:
  dependencies:
    - type: business_rule
      rule_id: rule_inventory_available
      relation: depends_on
    - type: system
      name: sqs
      component: order-events
      action: consume
  observability:
    datadog_monitor_ids:
      - "123"
      - "456"
    custom_metrics:
      name: routing_slip.orders.validated
      type: gauge
      tags:
        - env:production
        - team:backend
    log_markers:
      - order-validation
```

Lint rule: active business rules not yet covered by workflow should be warnings, not errors, because implementation may be incremental.

## Authoring Checklist

1. Start with `validate`.
2. Add `graphql_enrich` for anticorruption/read enrichment.
3. Use `assert`, `condition`, `cel` and `jump_if` for decisions.
4. Use `aws_action` or `rest_call` for side effects.
5. Add `audit`, `log` and optional `datadog_metric` at important milestones.
6. Use stable step ids and business comments.
7. Ensure idempotent side effects: stable message id, stable step ids and state store enabled in config.
8. For long workflows, split with `workflow_ref` and export a composed YAML only when needed.
