---
sidebar_position: 1
sidebar_label: "Visão geral"
---

# Routing Slip Pattern

De forma simplificada, um padrão `routing slip` (ou guia de encaminhamento) funciona como um documento de controle que acompanha uma mensagem através de uma série de etapas de processamento. Sua função fundamental é ditar a rota exata a ser seguida e registrar as ações concluídas em cada ponto de verificação.

No contexto de engenharia de software e microsserviços, o padrão `Routing Slip` é utilizado para orquestrar transações distribuídas de maneira descentralizada. Em vez de depender de um controlador central rígido ditando cada passo, **a própria mensagem carrega o seu itinerário**, contendo uma lista ordenada de serviços e regras que devem ser executados em sequência.

## Como funciona?

Aplicado rotineiramente em processos assíncronos, a mecânica é direta: cada serviço (ou *worker*) recebe a mensagem, executa sua tarefa de domínio isoladamente, atualiza o status no documento e encaminha o pacote para o próximo destino listado no roteiro.

Esse modelo descentralizado garante alto controle sobre o fluxo, gerando rastreabilidade, explicabilidade e viabilizando a criação de `checkpoints` duráveis. Isso assegura que, em caso de falha sistêmica, o processo possa ser retomado exatamente do ponto onde parou, sem repetir processamentos já consolidados.

```mermaid
flowchart LR
  %% Entrada
  A[API Gateway<br/>Recebe requisição]:::entry
  B[Routing Slip Creator<br/>Monta workflow + payload]:::control

  %% Canal
  EB[(EventBridge<br/>Message Channel)]:::channel

  %% Estado
  ST[(DynamoDB<br/>Execution State<br/>currentStep, status, history)]:::state
  LOG[(Datadog / CloudWatch<br/>Logs, Metrics, Traces)]:::obs
  DLQ[(SQS DLQ<br/>Falhas não recuperáveis)]:::error

  %% Steps
  S1["1. ValidateRequest"]:::step
  S2["2. ProcessBusinessLogic"]:::step
  S3["3. ExternalIntegration"]:::step
  S4["4. UpdateLedger/Database"]:::step
  S5["5. NotifyStakeholders"]:::step
  END["Workflow Finalizado"]:::success

  %% Fluxo principal
  A --> B
  B -->|"cria executionId<br/>routingSlip[1..5]"| ST
  B -->|publica step 1| EB

  EB -->|route: ValidateRequest| S1
  S1 -->|next: ProcessBusinessLogic| EB

  EB -->|route: ProcessBusinessLogic| S2
  S2 -->|next: ExternalIntegration| EB

  EB -->|route: ExternalIntegration| S3
  S3 -->|next: UpdateLedger| EB

  EB -->|route: UpdateLedger| S4
  S4 -->|next: NotifyStakeholders| EB

  EB -->|route: NotifyStakeholders| S5
  S5 --> END

  %% Estado e observabilidade
  S1 -. update step 1 .-> ST
  S2 -. update step 2 .-> ST
  S3 -. update step 3 .-> ST
  S4 -. update step 4 .-> ST
  S5 -. update step 5 .-> ST

  B -. log .-> LOG
  S1 -. trace/log .-> LOG
  S2 -. trace/log .-> LOG
  S3 -. trace/log .-> LOG
  S4 -. trace/log .-> LOG
  S5 -. trace/log .-> LOG

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
````

## Qual a proposta do framework?

O `routing-slip-pattern` é um framework low-code projetado para construir workflows modulares, altamente observáveis e com capacidade nativa de retomada (resume). Ele encapsula a complexidade do processamento distribuído, garantindo idempotência e mitigando riscos de duplicidade.

Com esta abstração, a equipe de engenharia pode:

- Declarar o fluxo de negócio de forma explícita em um arquivo YAML;
- Reutilizar _handlers_ padronizados para cada etapa;
- Enriquecer o payload dinamicamente com dados externos antes de aplicar regras de decisão;
- Registrar métricas estruturadas e logs distribuídos (traces) de forma transparente;
- Retomar a execução do exato ponto em que um erro transiente ocorreu, sem perder o estado.

A filosofia central é dar luz ao processo: **em vez de esconder a regra de negócio emaranhada em código imperativo, o workflow torna o caminho legível e auditável**.

## O Ecossistema

O framework atua em conjunto com outras ferramentas para entregar uma experiência completa de desenvolvimento e operação[cite: 2]:

- **Runtime**: O motor em Go que executa os workflows declarados[cite: 2].
- **Studio**: Uma interface visual para editar, simular e validar os fluxos localmente[cite: 2].
- **GraphQL Connector**: Uma fachada anti-corrupção para enriquecer payloads de forma padronizada[cite: 2].
- **Custom Business Metrics**: Ingestão e exibição das métricas e steps de falha[cite: 2].
- **State Store**: A camada de persistência que viabiliza a durabilidade do cursor e do payload[cite: 2].
- **MCP Gateway**: Ferramental avançado para validação semântica e planejamento assistido[cite: 2].