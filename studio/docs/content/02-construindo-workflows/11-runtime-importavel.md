---
sidebar_position: 11
sidebar_label: Runtime importável
---

# Runtime importável

O Studio é usado para arquitetar, validar e testar workflows. Em produção, a
aplicação importa o runtime, escolhe as origens da configuração e do workflow e
inicia o processamento no ambiente desejado.

## Instanciação com métricas

```go
agent, err := metrics.New(metrics.Config{
    ServiceEndpoint: "http://metrics-service:8080/v1/metrics",
})
go agent.Run(ctx)

runtime, err := routing.New(ctx, routing.Options{
    ConfigSource:   source.Local{Path: "config.yaml"},
    WorkflowSource: source.Local{Path: "workflow.yaml"},
    MetricsAgent:   agent,
})
```

## Origens suportadas

| Origem | Exemplo |
|---|---|
| Inline | `source.Inline(data)` |
| Local | `source.Local{Path: "workflow.yaml"}` |
| S3 | `source.AWS{Type: "s3", Region: "us-east-1", Bucket: "configs", Key: "workflow.yaml"}` |
| Secrets Manager | `source.AWS{Type: "secretsmanager", Region: "us-east-1", Name: "workflows/product"}` |
| Parameter Store | `source.AWS{Type: "ssm", Region: "us-east-1", Name: "/workflows/product"}` |
| DynamoDB | `source.AWS{Type: "dynamodb", Region: "us-east-1", Table: "workflows", Key: "product"}` |

O campo `Endpoint` permite usar LocalStack ou endpoints privados. Composição
por `workflow_ref` usa a mesma origem e resolve referências relativamente ao
workflow principal.

## Ambientes de execução

Para ECS, EKS, VM ou local:

```go
err := runtime.Run(ctx)
```

Para Lambda ou consumidores gerenciados pela aplicação:

```go
result, err := runtime.Process(ctx, payload)
```

Para ALB, API Gateway ou servidor existente:

```go
http.Handle("/process", runtime.Handler())
http.Handle("/mcp", runtime.MCPHandler())
```

## Idempotência e concorrência

O runtime usa `message_id_path` como identidade idempotente. Stores compatíveis
adquirem um lease antes do processamento, impedindo duas instâncias de executar
simultaneamente o mesmo evento. Snapshots concluídos são reutilizados e steps
finalizados não são repetidos quando a idempotência está habilitada.

## MCP e explicabilidade

O MCP server permite explicar o workflow, listar regras de negócio, recuperar
uma execução por `message_id` e buscar execuções por `correlation_id`,
`trace_id`, status ou workflow.
