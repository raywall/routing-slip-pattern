# Resumo da execucao

Ao final de cada simulacao, o Studio adiciona um resumo no fim da timeline.

O resumo mostra:

| Indicador | Descricao |
|---|---|
| Steps executados | Quantidade de etapas processadas. |
| Steps preservados | Etapas ja processadas antes do reprocessamento. |
| Steps pulados/parados | Etapas que encerraram ou desviaram o fluxo. |
| Erros | Falhas registradas durante a simulacao. |
| Tempo medio por step | Media de duracao das etapas executadas. |
| Integracoes API/servico | Total de chamadas ou simulacoes de integracao. |
| Tempo total | Duracao completa da simulacao. |
| Dif. tempo anterior | Diferenca entre o tempo do processamento atual e o anterior, quando for reprocessamento. |

As integracoes sao contabilizadas para handlers como `graphql_enrich`, `rest_call` e `notify`. Quando as integracoes reais estao desativadas, o resumo ainda registra as chamadas simuladas para deixar claro o desenho operacional do workflow.
