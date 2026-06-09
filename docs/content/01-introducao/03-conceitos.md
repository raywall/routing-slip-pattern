---
sidebar_position: 3
sidebar_label: "Conceitos fundamentais"
---

# Conceitos Fundamentais

O vocabulário abaixo é a base para o uso eficiente do Studio, estruturação dos YAMLs e compreensão da telemetria[cite: 4].

| Conceito | Definição no Ecossistema |
| :--- | :--- |
| **Workflow** | Sequência declarativa de etapas projetada para processar um fluxo de negócio[cite: 4]. |
| **Routing Slip** | O "itinerário" atachado à mensagem, contendo a lista ordenada de passos a executar[cite: 4]. |
| **Handler** | A unidade atômica de código (Go) projetada para uma responsabilidade única (ex: validar um esquema, fazer uma chamada HTTP, injetar logs estruturados)[cite: 4]. |
| **Connector** | A interface de entrada que trigga o workflow. Pode ser síncrona (REST `sync`) ou reativa (Kafka, SQS, SNS `async`)[cite: 4]. |
| **Payload** | O estado atual dos dados (JSON) que trafegam pelo fluxo, sendo enriquecidos ou mutados a cada etapa[cite: 4]. |
| **Cursor** | O ponteiro lógico salvo no banco de dados que indica qual é a próxima etapa pendente. Essencial para o *resume* de falhas[cite: 4]. |
| **State Store** | A camada de persistência (ex: DynamoDB) onde o *snapshot* contendo o payload, erros e o cursor é armazenado de forma segura[cite: 4]. |
| **Idempotência** | Mecanismo de proteção do framework que impede que um `handler` repita um efeito colateral (como um desconto financeiro) caso a etapa já conste como concluída no State Store[cite: 4]. |
| **Resiliência** | Políticas nativas configuráveis por step, englobando estratégias de *retry*, *exponential backoff* e *jitter*[cite: 4]. |
| **Rastreabilidade** | Propagação rigorosa de `trace_id` e `correlation_id` em toda a cadeia, ligando os *logs* aos painéis de APM (ex: Datadog)[cite: 4]. |
| **GraphQL Enrich** | Utilização da fachada anti-corrupção (`go-graphql-connector`) para compor o payload consultando bases de terceiros antes de decisões complexas[cite: 4]. |

## Anatomia da Execução

A relação entre os componentes segue uma lógica determinística[cite: 4]:

```mermaid
flowchart LR
  A[Payload de entrada] --> B[Workflow YAML]
  B --> C[Execução dos Handlers]
  C --> D[Payload Enriquecido]
  C --> E[(State Store)]
  C --> F[Métricas para APM]
  C --> G[Trace Propagado]
  H[Studio Local] --> B
  I[MCP Gateway] -. valida .-> B
````