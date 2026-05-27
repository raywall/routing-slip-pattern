# Interface

![Visao geral da interface do Studio](docs/images/studio-interface.jpg)

O Routing Slip Studio concentra criacao, revisao, teste e leitura da documentacao em uma unica tela. Ele foi desenhado para reduzir troca de contexto: voce escreve o YAML, ajusta payload, executa, le logs e consulta a documentacao sem sair do ambiente.

## Areas principais

| Area | Funcao |
| --- | --- |
| Painel lateral | Workspace, payload, configuracao e documentacao. |
| Editor central | Edicao do workflow YAML com lint, linhas, cores e atalhos. |
| Resultado | Logs de execucao, resumo, detalhes de etapas ou conteudo da documentacao. |

O Studio possui tema claro e escuro. O botao de tema fica na barra superior, ao lado do resumo da execucao.

## Ambientes

Quando publicado no GitHub Pages, o Studio pode mostrar um seletor de ambiente:

- **Producao**: `https://raywall.github.io/routing-slip-pattern`
- **Desenvolvimento**: `https://raywall.github.io/routing-slip-pattern/dev`

Isso permite consultar a documentacao estavel e a documentacao em evolucao sem misturar os dois contextos.

## Mobile

No celular, o Studio prioriza leitura. O ambiente de edicao e teste e ocultado e a documentacao passa a ter navegacao propria por menu lateral.

