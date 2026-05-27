# Objetivo

O objetivo do projeto e oferecer uma base para workflows resilientes, robustos, escalaveis, reutilizaveis, observaveis, seguros e modulares.

Na pratica, o framework ajuda a responder tres perguntas que aparecem em processos reais:

1. **O que deve acontecer agora?**
   O YAML descreve a proxima etapa, os parametros e as regras de decisao.

2. **O que aconteceu ate aqui?**
   Historico, metricas, traces e state store mostram cada etapa executada.

3. **Como continuar sem repetir o que ja foi feito?**
   O cursor salvo permite retomar do ponto correto e a idempotencia reduz risco de duplicar efeitos externos.

## Problema que o projeto resolve

Workflows longos costumam crescer em complexidade. Eles chamam APIs, validam regras, dependem de dados externos, tomam decisoes e precisam ser monitorados. Quando tudo isso fica espalhado em codigo, logs e configuracoes, fica mais dificil testar, explicar e reprocessar.

O Routing Slip organiza esse processo em uma sequencia declarativa e observavel.

## Resultado esperado

Ao usar o framework, o usuario deve conseguir:

- criar workflows sem escrever um orquestrador do zero;
- reutilizar handlers;
- testar cenarios no Studio;
- enriquecer payloads via GraphQL ou REST;
- saber onde uma execucao parou;
- reprocessar com seguranca;
- explicar o fluxo para outros times;
- evoluir o processo sem perder rastreabilidade.
