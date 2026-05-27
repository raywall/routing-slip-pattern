# Interface

![Visão geral da interface do Studio](docs/images/studio-interface.jpg)

O Routing Slip Studio concentra criação, revisão, teste e leitura da documentação em uma única tela. Ele foi desenhado para reduzir troca de contexto: voce escreve o YAML, ajusta payload, executa, le logs e consulta a documentação sem sair do ambiente.

## Areas principais

| Area | Função |
| --- | --- |
| Painel lateral | Workspace, payload, configuração e documentação. |
| Editor central | Edição do workflow YAML com lint, linhas, cores e atalhos. |
| Resultado | Logs de execução, resumo, detalhes de etapas ou conteúdo da documentação. |

O Studio possui tema claro e escuro. O botão de tema fica na barra superior, ao lado do resumo da execução.

## Ambientes

Quando publicado no GitHub Pages, o Studio pode mostrar um seletor de ambiente:

- **Produção**: `https://raywall.github.io/routing-slip-pattern`
- **Desenvolvimento**: `https://raywall.github.io/routing-slip-pattern/dev`

Isso permite consultar a documentação estável e a documentação em evolução sem misturar os dois contextos.

## Mobile

No celular, o Studio prioriza leitura. O ambiente de edição e teste e ocultado e a documentação passa a ter navegação própria por menu lateral.

