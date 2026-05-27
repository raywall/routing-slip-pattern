# Conceitos fundamentais

Antes de escrever workflows, vale alinhar alguns conceitos. Eles aparecem no Studio, nos arquivos YAML e nas metricas geradas durante a execucao.

| Conceito | O que significa neste projeto |
| --- | --- |
| Workflow | Sequencia declarativa de etapas que processa um payload. |
| Routing slip | Lista de passos que acompanha a mensagem e define o caminho de processamento. |
| Handler | Unidade de execucao. Cada handler sabe fazer uma coisa: validar, enriquecer, chamar API, auditar, saltar, filtrar. |
| Payload | Documento JSON que entra no workflow e vai sendo enriquecido ou transformado. |
| Cursor | Posicao da proxima etapa a executar. E o que permite retomar do ponto correto. |
| State store | Persistencia do snapshot de execucao: cursor, payload, historico, erros, trace e estado das etapas. |
| Idempotencia | Capacidade de evitar repetir um efeito externo quando uma etapa ja foi concluida. |
| Resiliencia | Politicas de retry, backoff, jitter e tratamento de falha por etapa. |
| Rastreabilidade | Uso de `trace_id`, `span_id`, `correlation_id` e historico para acompanhar uma execucao ponta a ponta. |
| Explicabilidade | Capacidade de entender por que uma etapa executou, parou, falhou, pulou ou redirecionou o fluxo. |
| GraphQL enrich | Enriquecimento do payload usando o `go-graphql-connector` como fachada para APIs e servicos externos. |
| Metricas | Eventos tecnicos e funcionais que alimentam dashboards e analises operacionais. |
| MCP | Interface de tools para agentes, Studio e automacoes consultarem, explicarem e planejarem workflows. |

## Como esses conceitos se conectam

```mermaid
flowchart LR
  A[Payload de entrada] --> B[Workflow YAML]
  B --> C[Handlers]
  C --> D[Payload enriquecido]
  C --> E[State store]
  C --> F[Metricas]
  C --> G[Trace]
  H[Studio] --> B
  I[MCP] --> B
  I --> E
  I --> F
```

O ponto central e simples: o workflow descreve o que precisa acontecer; o runtime executa; o state store permite retomar; as metricas e traces tornam tudo observavel; o Studio e o MCP ajudam a criar, validar e investigar.

