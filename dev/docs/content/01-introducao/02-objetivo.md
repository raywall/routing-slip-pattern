---
sidebar_position: 2
sidebar_label: "Objetivos e Motivações"
---

# Objetivos do Projeto

O objetivo fundamental deste projeto é fornecer uma fundação arquitetural resiliente, escalável e segura para workflows de missão crítica[cite: 3]. Na prática da operação distribuída, o framework soluciona três perguntas essenciais que frequentemente se tornam gargalos durante incidentes:

1. **O que deve acontecer agora?** O script YAML atua como a única fonte de verdade, definindo a próxima etapa e os parâmetros de decisão de forma puramente declarativa[cite: 3].
2. **O que já aconteceu até aqui?** O State Store aliado à injeção de trace_id em ferramentas de observabilidade revelam o histórico imutável de cada passo já consolidado[cite: 3].
3. **Como continuar de forma segura?** Em caso de indisponibilidade de uma API terceira, o *cursor* gravado garante que o processo seja retomado sem executar chamadas financeiras ou mutações não idempotentes novamente[cite: 3].

## O Problema que Resolvemos

Workflows de negócio tendem a crescer organicamente em complexidade[cite: 3]. Eles precisam consumir APIs REST, consultar bancos de dados, aprovar validações estritas de conformidade e emitir eventos[cite: 3]. Quando essas responsabilidades estão espalhadas em um código imperativo, acoplado e sem rastreabilidade nativa, tarefas como debugar uma falha, realizar um *retry* seguro ou explicar a regra de negócio atual se tornam custosas[cite: 3].

O `routing-slip-pattern` ataca esse problema isolando o **"O Quê"** (o fluxo no YAML) do **"Como"** (os *handlers* em Go), resultando em um código mais limpo e processos altamente testáveis[cite: 3].