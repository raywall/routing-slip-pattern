---
sidebar_position: 2
sidebar_label: "Objetivo"
---

# Objetivo do projeto

O projeto visa oferecer uma base sólida para criar workflows resilientes, robustos, escaláveis, reutilizáveis, observáveis, seguros e modulares, que na prática, ajuda a responder três perguntas que aparecem em processos reais:

1. **O que deve acontecer agora?**
   O YAML descreve a proxima etapa, os parâmetros e as regras de decisão.

2. **O que aconteceu ate aqui?**
   Histórico, métricas, traces e state store mostram cada etapa executada.

3. **Como continuar sem repetir o que já foi feito?**
   O cursor salvo permite retomar do ponto correto e a idempotência reduz risco de duplicar efeitos externos.

## Problema que o projeto resolve

Workflows longos costumam crescer em complexidade. Eles chamam APIs, validam regras, dependem de dados externos, tomam decisões e precisam ser monitorados. Quando tudo isso fica espalhado em código, logs e configurações, fica mais difícil testar, explicar e reprocessar.

O Routing Slip organiza esse processo em uma sequencia declarativa e observável.

## Resultado esperado

Ao usar o framework, o usuário deve conseguir:

- criar workflows sem escrever um orquestrador do zero;
- reutilizar handlers;
- testar cenários no Studio;
- enriquecer payloads via GraphQL ou REST;
- saber onde uma execução parou;
- reprocessar com segurança;
- explicar o fluxo para outros times;
- evoluir o processo sem perder rastreabilidade.
