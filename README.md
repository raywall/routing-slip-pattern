# Routing Slip em Go

Implementação do padrão de integração **Routing Slip** (Enterprise Integration Patterns)
em Go — modular, parametrizável e pronto para workflows reais.

> A proposta arquitetural expandida, incluindo enriquecimento externo via
> `go-graphql-connector`, métricas granulares inspiradas no
> `custom-business-metrics`, diagramas e exemplos, está em
> [DOCUMENTATION.md](DOCUMENTATION.md).

---

## Estrutura do projeto

```
routing-slip/
├── app/
│   ├── slip/
│   │   ├── slip.go          # Message, StepDef, Handler, Router, snapshots
│   │   ├── middleware.go    # LoggingMiddleware, RecoveryMiddleware
│   │   └── state_store.go   # StateStore e MemoryStateStore
│   ├── handlers/
│   │   └── handlers.go      # 6 handlers prontos
│   ├── config/
│   │   └── config.go        # Loader JSON -> []StepDef
│   ├── examples/
│   │   └── order_workflow.json
│   ├── main.go              # Demos, incluindo retomada por cursor
│   └── go.mod
├── go-graphql-connector/    # submodule privado para integracoes externas
├── custom-business-metrics/ # submodule para metricas e webview real-time
├── docker/                  # Dockerfiles customizaveis por servico
├── docker-compose.yml       # ambiente local integrado
├── Makefile
└── DOCUMENTATION.md
```

O submodule `go-graphql-connector` acompanha a branch `develop`.

---

## Conceitos-chave

### Message
Carrega o **payload** (dados mutáveis), **headers** e o **routing slip** embutido.
Thread-safe via `sync.RWMutex`. Mantém **histórico de execução** e **erros por etapa**.

### StepDef
Cada etapa tem um `Name` (resolve o handler registrado) e um mapa de `Params` arbitrários.

### Handler interface
```go
type Handler interface {
    Name() string
    Handle(ctx context.Context, msg *Message, params map[string]any) (proceed bool, err error)
}
```
Retornar `proceed=false` encerra o workflow graciosamente (sem erro).

### Router
- Mantém um **registry** de handlers por nome
- Aplica **middleware** (logging, recovery) em todos os handlers
- Suporta 3 **ErrorPolicy**: `StopOnError`, `ContinueOnError`, `SkipOnError`
- Respeita **context.Context** (cancelamento/timeout)

### Processamento retomável
- O `Router` aceita um `StateStore` plugável via `WithStateStore`.
- O estado da mensagem é persistido como `MessageSnapshot`.
- O snapshot guarda payload, headers, routing slip, histórico, erros e `cursor`.
- Em erro com `StopOnError`, o cursor volta para a etapa que falhou.
- Ao reprocessar, `MessageFromSnapshot` restaura a mensagem e o fluxo segue do ponto salvo.

### Evolução proposta
- Handler de enriquecimento externo via GraphQL para consultar APIs, DynamoDB,
  Redis, RDS, S3 e outros serviços por meio do `go-graphql-connector`.
- Middleware de métricas de negócio para registrar eventos por workflow, etapa,
  erro, decisão e enriquecimento em uma base como DynamoDB.
- Visualização real-time do processamento granular usando a ideia do
  `custom-business-metrics`.

### SlipBuilder (fluent API)
```go
steps := slip.NewSlip().
    Step("validate", map[string]any{"required": []string{"email"}}).
    Step("enrich",   map[string]any{"data": map[string]any{"source": "web"}}).
    Step("notify",   map[string]any{"channel": "email", "recipient": "ops@co"}).
    Build()
```

### Config JSON
```json
{
  "name": "meu-workflow",
  "error_policy": "continue",
  "steps": [
    { "name": "validate", "enabled": true, "params": { "required": ["id"] } },
    { "name": "enrich",   "enabled": false, "params": {} }
  ]
}
```
Etapas com `"enabled": false` são ignoradas na carga.

---

## Handlers incluídos

| Handler     | Função                                                    |
|-------------|-----------------------------------------------------------|
| `validate`  | Verifica campos obrigatórios no payload                   |
| `enrich`    | Injeta dados estáticos/computados no payload              |
| `transform` | Aplica operação a um campo (uppercase, prefix, suffix...) |
| `condition` | Para o workflow se condição não for atendida              |
| `notify`    | Simula envio de notificação (email/slack/webhook)         |
| `audit`     | Grava evento estruturado via slog                         |

---

## Executar

```bash
make submodules
make run
```

O `make run` executa uma suíte de cenários parametrizados:

- `payment-event-fulfillment`: evento de pagamento aprovado, consulta do pedido via GraphQL, emissão de nota fiscal, acionamento da expedição e baixa de estoque.
- `order-ok`: processamento completo com validação, enriquecimento GraphQL, decisão, transformação e auditoria.
- `order-stopped-by-decision`: payload enriquecido via GraphQL, mas represado por decisão funcional.
- `order-fail-and-resume`: falha técnica em uma etapa intermediária e reprocessamento a partir do cursor salvo.

## Testar

```bash
make test
```

## Ambiente local com Docker

```bash
make prepare
make run
```

Serviços expostos:

- Metrics Service: `http://localhost:8080`
- Metrics Webview: `http://localhost:5173`
- DynamoDB Local: `http://localhost:8000`
- Agent UDP: `localhost:8125`
- GraphQL Connector mock: `http://localhost:8090/graphql`
- External API mock: `http://localhost:8091`

O `make prepare` sobe DynamoDB, serviço de métricas, agent, webview, API externa mockada e um mock GraphQL que simula a função local do `go-graphql-connector` buscando dados nessa API externa. Em seguida, `make run` envia métricas para o dashboard e executa os workflows.

Os serviços do compose usam Dockerfiles explícitos em `docker/`. Eles incluem notas de customização para ambientes corporativos, como instalação de `ca-certificates`, caminho para CAs internas em `/usr/local/share/ca-certificates`, execução de `update-ca-certificates` e uso de `AWS_CA_BUNDLE` quando necessário.

Para parar:

```bash
make compose-down
```

---

## Criar um handler customizado

```go
type MyHandler struct{}

func (MyHandler) Name() string { return "my-step" }

func (MyHandler) Handle(ctx context.Context, msg *slip.Message, params map[string]any) (bool, error) {
    msg.Set("processed_by", "my-step")
    return true, nil
}

// Registrar:
router.MustRegister(MyHandler{})
```

---

## Adicionar middleware

```go
metricsMiddleware := func(next slip.Handler) slip.Handler {
    return &metricsWrapper{next: next}
}

router := slip.NewRouter(
    slip.WithMiddleware(slip.RecoveryMiddleware(), metricsMiddleware),
)
```
