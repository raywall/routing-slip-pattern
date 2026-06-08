---
name: go-graphql-connector-builder
description: Use when creating go-graphql-connector schemas, connectors, service.json, mock configs, GraphQL queries, Lambda/local bootstrap code, REST/DynamoDB/S3/RDS/Redis connector mappings, STS token authorization, retries, response transforms, partial responses, and CORS.
---

# Go GraphQL Connector Builder

Use this skill to create and review `go-graphql-connector` configurations and integration code.

Current module path:

```text
github.com/raywall/go-graphql-connector
```

## What The Connector Does

It builds a GraphQL API dynamically from:

- `service.json`
- `schema.json`
- `connectors.json`
- optional `mock.json`

It supports dynamic schema, field-level connectors, parallel resolution, REST, DynamoDB, Redis, S3, RDS, retries, timeouts, response transforms, partial response mode, local HTTP, ALB and API Gateway Lambda adapters, CORS and STS token authorization.

## service.json

```json
{
  "version": "1",
  "schema": "local:schema.json",
  "connectors": "local:connectors.json",
  "mock": "local:mock.json",
  "route": "/graphql",
  "pretty": true,
  "graphiql": true,
  "allow_partial": false,
  "authorization": {
    "require_token_sts": true,
    "skip_tls_verify": true,
    "tokenService": {
      "token_authorization_url": "https://mock.example.test/api/oauth/token",
      "headers": {
        "x-serial-number": "${EXTERNAL_API_SERIAL}"
      },
      "credentials": {
        "client_id": "${CLIENT_ID}",
        "client_secret": "${CLIENT_SECRET}"
      }
    }
  }
}
```

`schema`, `connectors`, `mock`, token URL and credentials can be inline or loaded from:

- `local:path`
- environment variable
- AWS SSM Parameter Store
- AWS Secrets Manager
- S3
- DynamoDB

## schema.json Pattern

Types:

```json
{
  "types": [
    {
      "name": "Order",
      "fields": [
        { "name": "id", "type": "String" },
        { "name": "status", "type": "String" },
        { "name": "total", "type": "Float" }
      ]
    },
    {
      "name": "WorkflowData",
      "fields": [
        { "name": "order", "type": "Object", "ofType": "Order" }
      ]
    }
  ],
  "query": {
    "name": "Query",
    "fields": [
      {
        "name": "dataSources",
        "type": "Object",
        "ofType": "WorkflowData",
        "args": [
          { "name": "orderID", "type": "String" }
        ]
      }
    ]
  }
}
```

Supported field kinds commonly used:

- `String`
- `Int`
- `Float`
- `Boolean`
- `Object` with `ofType`
- `List` with `ofType`

## connectors.json REST Pattern

```json
{
  "connectors": [
    {
      "field": "order",
      "adapter": "rest",
      "adapterConfig": {
        "baseUrl": "${EXTERNAL_API_URL}",
        "endpoint": "/ecommerce/v1/orders/{orderID}",
        "method": "GET",
        "skipTLSVerify": true,
        "headers": {
          "X-Serial-Number": "${EXTERNAL_API_SERIAL}"
        }
      },
      "responseTransform": {
        "unwrapPath": "data",
        "errorsPath": "errors",
        "failOnErrors": true
      },
      "timeoutMs": 2000,
      "retries": 1
    }
  ]
}
```

Guidance:

- `field` must match a field under `dataSources` output type.
- Use `{argName}` placeholders for GraphQL query args.
- Use `${ENV_VAR}` for environment values.
- Use `responseTransform.unwrapPath` when external APIs wrap payloads in `data`.
- Keep `errorsPath` when API returns `errors`; use `failOnErrors` for strict mode.
- Use retries/timeouts/circuit breaker when external dependencies are fragile.

## Query Pattern

```graphql
query ($orderID: String!) {
  dataSources(orderID: $orderID) {
    order {
      id
      status
      total
    }
  }
}
```

Request body:

```json
{
  "query": "query ($orderID: String!) { dataSources(orderID: $orderID) { order { id status total } } }",
  "variables": {
    "orderID": "ORD-1001"
  }
}
```

The connector may remove the outer GraphQL `data` wrapper depending on configured handler behavior, but GraphQL clients normally expect `data.dataSources`.

## Local/Lambda Bootstrap

Typical local/Lambda entrypoint:

```go
config, err := graphql.LoadConfig("config/service.json")
if err != nil { panic(err) }

resources := &cloud.CloudContextList{
    cloud.SSMContext,
    cloud.SecretsManagerContext,
}

api, err := graphql.New(config, resources, "us-east-1", "http://localhost:4566")
if err != nil { panic(err) }

handler := api.NewHandler(config.Pretty, graphql.CORSFromEnv())
http.Handle(api.Config.Route, handler)
```

For Lambda ALB/API Gateway, wrap the handler with connector adapters available in the installed version.

## STS Token Rules

When an API requires a managed token:

- Set `authorization.require_token_sts: true`.
- Configure `tokenService.token_authorization_url`.
- Configure token headers and credentials.
- Use `authorization.skip_tls_verify: true` only for private/test environments where certificate validation cannot work.
- Prefer env/secret references for `client_secret`.

## Output Checklist

When asked to create a connector setup, produce:

1. `service.json`
2. `schema.json`
3. `connectors.json`
4. optional `mock.json`
5. GraphQL query examples
6. curl/Bruno request body
7. local/Lambda bootstrap guidance
8. notes about `unwrapPath`, `errorsPath`, retries, timeout and token auth
