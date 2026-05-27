# Recursos de um conector

O `go-graphql-connector` permite compor uma fachada de dados sem espalhar logica de integração dentro do workflow.

## Transformação de resposta

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

## Backoff, jitter e circuit breaker

Além do retry simples, cada connector pode definir uma politica de resiliência:

```json
{
  "timeoutMs": 1500,
  "retries": 2,
  "resilience": {
    "backoff": "exponential",
    "initial_interval_ms": 100,
    "max_interval_ms": 1000,
    "jitter": true,
    "circuit_breaker": {
      "enabled": true,
      "failure_threshold": 5,
      "open_timeout_ms": 30000
    }
  }
}
```

O circuit breaker evita insistir em uma integração que já demonstrou estar indisponível. Enquanto o circuito estiver aberto, o connector falha rapidamente com a classe `circuit_open`, permitindo que o workflow aplique fallback, retry posterior ou reprocessamento.

## Token STS

O conector suporta emissão de token STS/OAuth por `client_credentials`. Quando habilitado, o runtime:

1. chama o endpoint definido em `token_authorization_url`;
2. envia `grant_type=client_credentials`, `client_id` e `client_secret`;
3. aplica headers customizados do `tokenService.headers`;
4. guarda o `access_token` retornado;
5. injeta `Authorization: Bearer <token>` nos connectors REST.

```json
{
  "authorization": {
    "require_token_sts": true,
    "skip_tls_verify": false,
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

### Opções

| Campo | Descrição |
|---|---|
| `require_token_sts` | Liga/desliga autenticação centralizada. |
| `skip_tls_verify` | Desativa validação TLS do emissor de token quando `true`. Use apenas em ambientes privados controlados. |
| `token_authorization_url` | URL do emissor de token. |
| `headers` | Headers enviados somente para a chamada de token. |
| `credentials.client_id` | Identificador da aplicação cliente. |
| `credentials.client_secret` | Segredo da aplicação cliente. |

### Quando usar

Use token STS quando varias APIs integradas exigem o mesmo `Bearer token`. O workflow continua chamando apenas GraphQL, enquanto o conector gerencia autenticação e renovação.

```mermaid
flowchart LR
    Workflow[Workflow] --> GraphQL[GraphQL Connector]
    GraphQL --> Token[Token Service]
    Token --> GraphQL
    GraphQL -->|Authorization Bearer| API1[API de catalogo]
    GraphQL -->|Authorization Bearer| API2[API de entrega]
```

### Cuidados

- Se um connector REST ja possuir `Authorization` em `adapterConfig.headers`, o token automático nao sobrescreve esse valor.
- Mantenha `client_secret` em Secrets Manager, SSM seguro ou variável de ambiente protegida.
- Prefira instalar a CA interna na imagem do container. Use `skip_tls_verify` como exceção temporária ou para laboratórios controlados.
- Configure `timeoutMs` e `retries` por fonte para evitar que uma API lenta degrade todo o workflow.
