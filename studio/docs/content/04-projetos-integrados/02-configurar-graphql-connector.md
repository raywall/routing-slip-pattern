# Configurar GraphQL Connector

O conector usa tres configuracoes principais:

| Arquivo | Finalidade |
|---|---|
| `service.json` | Define schema, connectors, mock, rota e autorizacao. |
| `schema.json` | Descreve tipos, campos e argumentos GraphQL. |
| `connectors.json` | Mapeia campos GraphQL para adapters externos. |

Exemplo de `service.json` local:

```json
{
  "schema": "local:schema.json",
  "connectors": "local:connectors.json",
  "mock": "local:mock.json",
  "route": "/graphql",
  "pretty": true,
  "graphiql": true,
  "allow_partial": false
}
```

Exemplo de connector REST:

```json
{
  "connectors": [
    {
      "field": "catalogo",
      "adapter": "rest",
      "adapterConfig": {
        "baseUrl": "https://mock.raysouz.studio",
        "endpoint": "/catalogo/produtos/{sku}",
        "method": "GET",
        "headers": {
          "x-correlation-id": "{correlation_id}"
        }
      },
      "keyPattern": "/catalogo/produtos/{sku}",
      "timeoutMs": 3000,
      "retries": 1,
      "responseTransform": {
        "unwrapPath": "data",
        "errorsPath": "errors",
        "failOnErrors": true
      }
    }
  ]
}
```

Fontes suportadas pelos paths de configuracao:

| Prefixo | Uso |
|---|---|
| `local:` | Arquivo local relativo ao service.json. |
| `env:` | Variavel de ambiente. |
| `ssm:` | AWS Systems Manager Parameter Store. |
| `secret:` ou `secrets:` | AWS Secrets Manager. |
| `s3:` | Objeto no S3. |

Adapters suportados no projeto:

- `rest`;
- `dynamodb`;
- `s3`;
- `rds`;
- `redis`.
