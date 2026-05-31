---
sidebar_position: 5
sidebar_label: "Capacidades e benefícios"
---

# Capacidades e benefícios

O Routing Slip Pattern foi pensado para workflows que precisam ser robustos, reutilizáveis, observáveis e fáceis de evoluir. Ele e especialmente util quando o processo possui varias etapas, depende de dados externos ou precisa ser reprocessado com segurança.

## O que voce consegue fazer

| Capacidade | Beneficio pratico |
| --- | --- |
| Workflow em YAML | Regras e etapas ficam visíveis, versionáveis e revisáveis. |
| Handlers reutilizáveis | O mesmo bloco de comportamento atende vários domínios. |
| Enriquecimento externo | Dados de APIs e serviços entram no payload antes das decisões. |
| State store | Falhas nao obrigam recomeçar tudo. O cursor indica de onde continuar. |
| Idempotência por etapa | Evita repetir efeitos externos ja concluídos. |
| Resiliência por etapa | Retry, backoff, jitter e fallback ficam declarados no workflow. |
| Observabilidade granular | Cada etapa gera histórico, métricas e trace. |
| Composição de scripts | Workflows grandes podem ser divididos em arquivos menores. |
| Studio | Editor, lint, payload, simulação, logs e documentação no mesmo lugar. |
| MCP | Tools para validar, explicar, consultar estado e planejar workflows. |

## Problemas que o framework ajuda a resolver

- processamento que falha no meio e precisa continuar sem repetir etapas anteriores;
- integrações externas instáveis que precisam de retry e circuit breaker;
- fluxos longos que ficam difíceis de entender quando escritos em código imperativo;
- falta de visibilidade sobre qual etapa falhou e por que falhou;
- duplicação de logica de validação, enriquecimento e auditoria;
- dificuldade para testar diferentes cenários de entrada;
- necessidade de explicar o comportamento de uma execução para outros times.

## Exemplo de retorno operacional

Um fluxo de venda online pode validar o evento, consultar o pedido, reservar entrega, atualizar estoque e notificar o cliente. Se a notificação falhar, o state store preserva o cursor e o payload enriquecido. O reprocessamento continua da etapa correta e as métricas mostram o que foi executado, o que falhou e quanto tempo cada etapa levou.

Esse modelo reduz retrabalho, evita efeitos duplicados e facilita investigação.

