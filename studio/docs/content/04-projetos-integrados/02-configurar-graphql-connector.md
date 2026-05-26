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

## Autenticacao STS para APIs REST

Quando `authorization.require_token_sts` e `true`, o `go-graphql-connector` emite um token usando `client_credentials` e injeta automaticamente `Authorization: Bearer <token>` nas chamadas dos connectors REST que nao tenham um header `Authorization` configurado manualmente.

```json
{
  "schema": "local:schema.json",
  "connectors": "local:connectors.json",
  "mock": "local:mock.json",
  "route": "/graphql",
  "pretty": true,
  "graphiql": true,
  "allow_partial": false,
  "authorization": {
    "require_token_sts": true,
    "tokenService": {
      "token_authorization_url": "${STS_TOKEN_URL}",
      "headers": {
        "x-serial-number": "${STS_SERIAL_NUMBER}"
      },
      "credentials": {
        "client_id": "${STS_CLIENT_ID}",
        "client_secret": "${STS_CLIENT_SECRET}"
      }
    }
  }
}
```

Campos suportados em `authorization`:

| Campo | Obrigatorio | Descricao |
|---|---:|---|
| `require_token_sts` | Sim | Ativa ou desativa emissao de token. |
| `tokenService.token_authorization_url` | Sim | Endpoint OAuth/STS usado para emitir o token. Aceita `local`, `env`, `ssm`, `secret`, `s3`, `dynamodb` ou `${VAR}` inline. |
| `tokenService.headers` | Nao | Headers enviados para o emissor do token, como `x-serial-number`. Os nomes sao preservados. |
| `tokenService.credentials.client_id` | Sim | Client id, podendo vir inline, env, SSM ou Secrets Manager. |
| `tokenService.credentials.client_secret` | Sim | Client secret, podendo vir inline, env, SSM ou Secrets Manager. |

Exemplo com o mock publico:

```bash
export STS_TOKEN_URL="https://mock.raysouz.studio/api/oauth/token"
export STS_SERIAL_NUMBER="b7af3a9e-6d1a-4b15-9837-3e0f0b47e5b4"
export STS_CLIENT_ID="f47ac10b-58cc-4372-a567-0e02b2c3d479"
export STS_CLIENT_SECRET="550e8400-e29b-41d4-a716-446655440000"
```

> O token e cacheado ate perto da expiracao. Uma nova chamada so e feita quando nao ha token ou quando ele esta proximo de expirar.

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

## Sintaxe dos connectors

Cada item de `connectors.json` liga um campo GraphQL a uma fonte externa.

| Campo | Obrigatorio | Descricao |
|---|---:|---|
| `field` | Sim | Nome do campo GraphQL que sera resolvido. |
| `adapter` | Sim | Tipo de fonte: `rest`, `dynamodb`, `s3`, `rds` ou `redis`. |
| `adapterConfig` | Sim | Configuracao especifica do adapter. |
| `keyPattern` | Condicional | Template usado como chave/endpoint/query. Para REST, se omitido, usa `adapterConfig.endpoint`. |
| `timeoutMs` | Nao | Timeout por chamada. Padrao: 3000 ms. |
| `retries` | Nao | Tentativas adicionais apos falha. |
| `optional` | Nao | Se `true`, falhas da fonte nao derrubam a query. |
| `responseTransform` | Nao | Normaliza respostas com wrappers como `data` e `errors`. |

Variaveis entre `{}` sao substituidas pelos argumentos da query GraphQL:

```json
{
  "field": "customer",
  "adapter": "rest",
  "adapterConfig": {
    "baseUrl": "https://mock.raysouz.studio",
    "endpoint": "/customers/{customerID}",
    "method": "GET",
    "headers": {
      "x-serial-number": "${EXTERNAL_API_SERIAL}"
    }
  },
  "timeoutMs": 1500,
  "retries": 1
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
