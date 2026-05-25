# Recursos do conector

O `go-graphql-connector` permite compor uma fachada de dados sem espalhar logica de integracao dentro do workflow.

## Transformacao de resposta

Use `responseTransform.unwrapPath` para retornar somente o trecho relevante:

```json
{
  "responseTransform": {
    "unwrapPath": "data",
    "errorsPath": "errors",
    "failOnErrors": true
  }
}
```

Isso reduz payloads como:

```json
{ "data": { "id": "P-100" }, "errors": [] }
```

para:

```json
{ "id": "P-100" }
```

## Falha parcial

`allow_partial` no service ou `optional` por connector permite que uma fonte falhe sem derrubar toda a query, quando isso fizer sentido para o processo.

## Timeout e retry

```json
{
  "timeoutMs": 3000,
  "retries": 2
}
```

## Token STS

A configuracao de autorizacao ainda existe no conector:

```json
{
  "authorization": {
    "require_token_sts": true,
    "tokenService": {
      "token_authorization_url": "env:STS_TOKEN_URL",
      "Credentials": {
        "client_id": "env:STS_CLIENT_ID",
        "client_secret": "secrets:/graphql/dev/credentials:json"
      }
    }
  }
}
```

Observacao: a configuracao cria o gerenciador de token. Se uma API REST precisar receber `Authorization: Bearer <token>`, confirme no runtime se o adapter esta injetando o token automaticamente ou se sera necessario plugar esse token nos headers do connector.
