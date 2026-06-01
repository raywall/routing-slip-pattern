---
sidebar_position: 1
sidebar_label: "Visão geral"
---

# Visão geral do projeto

O `routing-slip-pattern` é um framework low-code versátil e robusto, que permite a construção de workflows modulares, rastreáveis, reprocessáveis e seguros. Na prática, ele considera cada processamento como algo único, garantindo a idempotência e mitigando riscos de duplicidade, além de tornar possível: 

- Descrever o fluxo de processamento em formato YAML;
- Executar cada etapa com handlers reutilizáveis;
- Enriquecer o payload com dados externos;
- Registrar métricas e logs;
- Retomar o processamento do ponto correto quando algo falha;

A proposta é bastante simples: `em vez de esconder o processo dentro de código imperativo, o workflow deixa o caminho visível`. Cada etapa tem nome, parâmetros, entrada, saída, status e histórico.

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

O framework foi desenhado para atender qualquer domínio: pedidos, catalogo, atendimento, logística, onboarding, validação documental, notificações, sincronização de dados ou qualquer processo que precise ser realizado em etapas.

## O que voce encontra no ecossistema?

- **Runtime** para executar workflows.
- **Studio** para editar, validar, simular e entender scripts.
- **GraphQL Connector** para enriquecer payloads com APIs e serviços externos.
- **Custom Business Metrics** para visualizar execuções, etapas e falhas.
- **State store** para retomada robusta.
- **MCP Gateway** para tools de validação, explicação, consulta de estado e planejamento assistido.

## Recomendações

1. Leia os conceitos fundamentais.
2. Abra a estrutura básica de workflow.
3. Entenda paths e arrays.
4. Explore os handlers.
5. Use o Studio para editar e executar um exemplo.
6. Ligue integrações reais quando o fluxo local estiver claro.
