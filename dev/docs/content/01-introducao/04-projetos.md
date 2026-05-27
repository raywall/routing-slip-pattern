# Projetos do ecossistema

O framework e formado por projetos independentes, mas pensados para funcionar juntos. Essa separacao permite usar apenas o que for necessario em cada contexto.

## routing-slip-pattern

E o runtime de workflows. Ele carrega um YAML, recebe eventos por REST, Kafka ou SQS, executa handlers, persiste estado, emite metricas e permite retomada a partir do cursor salvo.

Use quando precisar:

- orquestrar processos com varias etapas;
- enriquecer payloads antes de decidir;
- reprocessar do ponto de falha;
- tornar o fluxo explicavel para operacao e engenharia;
- reutilizar os mesmos handlers em fluxos diferentes.

## go-graphql-connector

E a camada de integracao para APIs e servicos externos. Ele permite criar uma API GraphQL configuravel que consulta REST, DynamoDB, Redis, S3, RDS e outros conectores suportados.

Use quando precisar:

- concentrar consultas externas em uma interface GraphQL;
- reduzir acoplamento entre workflow e APIs internas;
- reutilizar configuracoes de connector em varios workflows;
- gerenciar token, timeout, retry e circuit breaker por integracao;
- simplificar enriquecimentos usando `graphql_enrich`.

## custom-business-metrics

E a camada de metricas e dashboards. Ela recebe eventos do runtime, armazena dados e permite acompanhar execucoes, etapas, falhas, duracao, retries, traces e reprocessamentos.

Use quando precisar:

- acompanhar workflows em tempo real;
- buscar processos por `correlation_id`, `trace_id` ou tags;
- entender gargalos por etapa;
- comparar execucao original e reprocessamento;
- criar dashboards especificos por dominio.

## Como os projetos trabalham juntos

```mermaid
sequenceDiagram
  participant Event as Evento
  participant Runtime as routing-slip-pattern
  participant GraphQL as go-graphql-connector
  participant API as APIs externas
  participant Metrics as custom-business-metrics

  Event->>Runtime: payload
  Runtime->>GraphQL: graphql_enrich
  GraphQL->>API: consulta APIs/servicos
  API-->>GraphQL: dados
  GraphQL-->>Runtime: payload enriquecido
  Runtime->>Metrics: eventos por etapa
  Runtime->>Runtime: state store + cursor
```

