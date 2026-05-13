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
├── slip/
│   ├── slip.go          # Message, StepDef, Handler interface, Router, SlipBuilder
│   └── middleware.go    # LoggingMiddleware, RecoveryMiddleware
├── handlers/
│   └── handlers.go      # 6 handlers prontos: validate, enrich, transform, condition, notify, audit
├── config/
│   └── config.go        # Loader JSON → []StepDef
├── examples/
│   └── order_workflow.json  # Workflow de pedido (configurado em JSON)
├── main.go              # 5 demos comentados
└── go.mod
```

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
go run .
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
# routing-slip-pattern
