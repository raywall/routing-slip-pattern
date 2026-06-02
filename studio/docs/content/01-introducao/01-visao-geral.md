---
sidebar_position: 1
sidebar_label: "Visão geral"
---

# Routing Slip Pattern

De forma simplificada, um padrão `routing slip` (ou guia de encaminhamento) pode ser definido como um documento ou registro que acompanha um item, mensagem ou paciente através de uma série de etapas ou departamentos de processamento. Tendo como função fundamental, ditar a rota exata a ser seguida e registrar as ações concluídas em cada ponto.

Apesar de ser um termo que pode ser aplicado nos três contextos distintos abaixo, vamos focar apenas no uso do padrão voltado aplicado ao desenvolvimento de sistemas.

  1. Arquitetura de Software (mensageria)
  2. Ambientes Médicos e Clínicos
  3. Logística, Controle de Produção e Corporativo

O padrão `Routing Slip` (ou padrão de encaminhamento) é usado para criar transações distribuídas, permitindo que em vez de um controlador central ditando o caminho (orquestração), **a mensagem carrega seu próprio "itinerário"** contendo uma lista de serviços (atividades) que devem ser executados em sequência.


## Como funciona?

Bastante comum em processos assíncronos ou microsserviços, tanto a ideia quanto a estrutura aplicada a um `routing slip` é bastante simples. `Cada serviço recebe a mensagem, executa sua tarefa, atualiza o status no documento e encaminha para o próximo destino listado`, possibilitando assim um maior controle do processo. Além de gerar rastreabilidade, explicabilidade e viabilizar a criação de `checkpoints` que garantem que um processo possa ser retomado a qualquer momento do exato ponto onde parou. 

```mermaid
flowchart LR
  %% Entrada
  A[API Gateway<br/>Recebe proposta]:::entry
  B[Routing Slip Creator<br/>Monta workflow + payload]:::control

  %% Canal
  EB[(EventBridge<br/>Message Channel)]:::channel

  %% Estado
  ST[(DynamoDB<br/>Execution State<br/>currentStep, status, history)]:::state
  LOG[(CloudWatch / Datadog<br/>Logs, Metrics, Traces)]:::obs
  DLQ[(SQS DLQ<br/>Falhas não recuperáveis)]:::error

  %% Steps
  S1["1. ValidateCustomer Lambda"]:::step
  S2["2. CalculateOffer ECS/Fargate"]:::step
  S3["3. FraudCompliance Lambda"]:::step
  S4["4. FormalizeContract ECS/Lambda"]:::step
  S5["5. NotifyCustomer Lambda/SNS"]:::step
  END["Workflow Finalizado"]:::success

  %% Fluxo principal
  A --> B
  B -->|"cria executionId<br/>routingSlip[1..5]"| ST
  B -->|publica step 1| EB

  EB -->|route: ValidateCustomer| S1
  S1 -->|next: CalculateOffer| EB

  EB -->|route: CalculateOffer| S2
  S2 -->|next: FraudCompliance| EB

  EB -->|route: FraudCompliance| S3
  S3 -->|next: FormalizeContract| EB

  EB -->|route: FormalizeContract| S4
  S4 -->|next: NotifyCustomer| EB

  EB -->|route: NotifyCustomer| S5
  S5 --> END

  %% Estado e observabilidade
  S1 -. update step 1 .-> ST
  S2 -. update step 2 .-> ST
  S3 -. update step 3 .-> ST
  S4 -. update step 4 .-> ST
  S5 -. update step 5 .-> ST

  B -. log .-> LOG
  S1 -. log/trace .-> LOG
  S2 -. log/trace .-> LOG
  S3 -. log/trace .-> LOG
  S4 -. log/trace .-> LOG
  S5 -. log/trace .-> LOG

  %% Falhas
  S1 -->|erro/retry excedido| DLQ
  S2 -->|erro/retry excedido| DLQ
  S3 -->|erro/retry excedido| DLQ
  S4 -->|erro/retry excedido| DLQ
  S5 -->|erro/retry excedido| DLQ

  %% Agrupamentos visuais
  subgraph AWS_Entry[Entrada]
    A
    B
  end

  subgraph AWS_Routing[Message Routing]
    EB
  end

  subgraph AWS_Workers[Execução do Routing Slip]
    S1
    S2
    S3
    S4
    S5
  end

  subgraph AWS_State[Estado, Timeline e Observabilidade]
    ST
    LOG
  end

  subgraph AWS_Failure[Tratamento de Falhas]
    DLQ
  end

  %% Estilos
  classDef entry fill:#FFF3B0,stroke:#D97706,stroke-width:2px,color:#111827;
  classDef control fill:#DBEAFE,stroke:#2563EB,stroke-width:2px,color:#111827;
  classDef channel fill:#E0E7FF,stroke:#4F46E5,stroke-width:2px,color:#111827;
  classDef step fill:#DCFCE7,stroke:#16A34A,stroke-width:2px,color:#111827;
  classDef state fill:#F3E8FF,stroke:#9333EA,stroke-width:2px,color:#111827;
  classDef obs fill:#E0F2FE,stroke:#0284C7,stroke-width:2px,color:#111827;
  classDef error fill:#FEE2E2,stroke:#DC2626,stroke-width:2px,color:#111827;
  classDef success fill:#BBF7D0,stroke:#15803D,stroke-width:3px,color:#111827;
```

[ref](https://www.enterpriseintegrationpatterns.com/patterns/messaging/RoutingTable.html)

## Qual a proposta do framework?

O `routing-slip-pattern` é um framework low-code versátil, robusto e altamente customizável para construir workflows modulares, rastreáveis e capazes de retomada. Ele trata cada processamento como uma execução única, garantindo idempotência e reduzindo riscos de duplicidade. Com ele, você pode:

- Descrever o fluxo de processamento em formato YAML;
- Executar cada etapa com handlers reutilizáveis;
- Enriquece o payload com dados externos;
- Registrar métricas e logs;
- Retomar o processamento exatamente do ponto onde ocorreu uma falha.

A proposta é simples: **em vez de esconder o processo dentro de código imperativo, o workflow deixa o caminho visível**. Cada etapa tem nome, parâmetros, entrada, saída, status e histórico.

```mermaid
flowchart LR
  A[Evento REST/Kafka/SQS] --> B[Workflow YAML]
  B --> C[Handlers]
  C --> D[Payload atualizado]
  C --> E[State store]
  C --> F[Métricas e traces]
  G[Studio] --> B
  H[MCP] --> B
```

O framework foi projetado para atender qualquer domínio: pedidos, catálogo, atendimento, logística, onboarding, validação documental, notificações, sincronização de dados – ou qualquer processo que precise ser executado em etapas.

## O que você encontra no ecossistema?

- **Runtime**: executa os workflows.
- **Studio**: editor para criar, validar, simular e entender scripts.
- **GraphQL Connector**: enriquece payloads com APIs e serviços externos.
- **Custom Business Metrics**: visualiza execuções, etapas e falhas.
- **State store**: permite retomada robusta.
- **MCP Gateway**: fornece ferramentas para validação, explicação, consulta de estado e planejamento assistido.

## Recomendações

1. Leia os [conceitos fundamentais](#).
2. Conheça a estrutura básica de um workflow.
3. Entenda como funcionam paths e arrays.
4. Explore os handlers disponíveis.
5. Utilize o Studio para editar e executar um exemplo.
6. Integre sistemas reais somente quando o fluxo local estiver claro.
