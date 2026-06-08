# Skills do Ecossistema de Workflows

Este catálogo reúne skills para orientar modelos como Devin, Claude, Codex ou outros agentes na criação de soluções baseadas em:

- `routing-slip-pattern`
- `go-graphql-connector`
- `custom-business-metrics`

A proposta é acelerar a criação de workflows, configurações, conectores, schemas, queries, observabilidade e bootstraps de aplicação.

## Skills Criadas

| Skill | Pasta | Quando usar |
|---|---|---|
| Routing Slip Workflow Author | `routing-slip-workflow-author` | Criar, revisar, compor ou explicar workflows YAML do `routing-slip-pattern`. |
| Routing Slip Runtime Bootstrap | `routing-slip-runtime-bootstrap` | Gerar `config.yaml`, `main.go`, Lambda handler, configuração de trigger, state store, idempotência, métricas e MCP. |
| Go GraphQL Connector Builder | `go-graphql-connector-builder` | Criar `service.json`, `schema.json`, `connectors.json`, mocks, queries GraphQL e bootstrap local/Lambda do conector. |
| Custom Business Metrics Observability | `custom-business-metrics-observability` | Instrumentar métricas, desenhar dashboards, criar queries de widgets, usar filtros, trace e correlation id. |
| Workflow Ecosystem Architecture | `workflow-ecosystem-architecture` | Projetar uma solução ponta a ponta usando os três projetos de forma integrada. |

## Como Aplicar

Cada pasta contém um `SKILL.md` autocontido. Para usar com um modelo:

1. Abra o `SKILL.md` da skill adequada.
2. Forneça o conteúdo como contexto inicial do agente.
3. Inclua a demanda concreta do usuário.
4. Peça que o agente gere artefatos diretamente: YAML, JSON, Go, queries, payloads, comandos e checklist de validação.

## Sugestões de Uso

### Criar um workflow

Use:

```text
skills/routing-slip-workflow-author/SKILL.md
```

Prompt sugerido:

```text
Usando esta skill, crie um workflow routing-slip-pattern para processar evento de pedido confirmado.
O fluxo deve validar o payload, consultar dados do pedido via GraphQL, reservar estoque, publicar evento SQS e auditar o resultado.
```

### Criar uma aplicação runtime

Use:

```text
skills/routing-slip-runtime-bootstrap/SKILL.md
```

Prompt sugerido:

```text
Usando esta skill, gere config.yaml, main.go e payloads de teste para executar um workflow via REST sync em :8088, com state store em DynamoDB Local, idempotência e métricas habilitadas.
```

### Criar configuração GraphQL

Use:

```text
skills/go-graphql-connector-builder/SKILL.md
```

Prompt sugerido:

```text
Usando esta skill, gere schema.json, connectors.json, service.json e uma query GraphQL para consultar order, customer, inventory e deliveryPolicy a partir de APIs REST externas.
```

### Criar observabilidade

Use:

```text
skills/custom-business-metrics-observability/SKILL.md
```

Prompt sugerido:

```text
Usando esta skill, desenhe métricas, tags, queries de widgets e um dashboard operacional para acompanhar sucesso, falha, reprocessamento, duração p95 e erros por etapa de um workflow.
```

### Projetar uma solução completa

Use:

```text
skills/workflow-ecosystem-architecture/SKILL.md
```

Prompt sugerido:

```text
Usando esta skill, proponha a arquitetura e os artefatos base para uma solução com routing-slip-pattern, go-graphql-connector e custom-business-metrics, incluindo estrutura de pastas, configs, workflows, queries, dashboards e testes.
```

## Diretrizes Gerais

- Mantenha exemplos neutros, como pedidos, catálogo, inventário, expedição, notificações e documentos.
- Não use domínios sensíveis nos exemplos.
- Ao gerar Go, considere que os projetos poderão estar disponíveis no `pkg.go.dev`, mas confirme a API pública instalada antes de usar nomes de construtores concretos.
- Quando a API pública ainda não estiver clara, gere pseudo-código com comentários `TODO` e explique o ponto de adaptação.
- Preserve comentários existentes em workflows.
- Regras de negócio ativas sem cobertura no script devem gerar aviso, não erro, pois a implementação pode evoluir em etapas.
- Use `correlation_id` único e estável por processo de negócio.
- Use `trace_id` para rastreabilidade técnica distribuída.
- Prefira `graphql_enrich` para leituras/enriquecimento e `aws_action`/`rest_call` para comandos e efeitos externos.
