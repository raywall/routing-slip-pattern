![Logs de execução das etapas](docs/images/studio-result.jpg#right-table)

A area de resultado mostra a execucao do workflow em ordem cronologica. Cada grupo representa uma fase do processamento, normalmente associada a uma etapa do YAML.

Dentro de uma fase voce pode encontrar logs de validacao, enriquecimento, simulacao de integracao, decisao, erro ou conclusao. A leitura por fase ajuda a responder perguntas bem praticas:<br>

- qual etapa foi executada;
- qual entrada foi usada;
- qual regra foi aplicada;
- qual saida foi produzida;
- onde o processamento parou;
- o que precisa ser corrigido antes de reexecutar.

Quando um log corresponde a uma etapa do YAML, clicar nele leva o foco para o trecho equivalente no editor. Isso e especialmente util em scripts longos, quando procurar manualmente pela etapa quebraria o ritmo da analise.

Ao final de cada simulacao, o Studio adiciona um resumo no fim da timeline. Ele serve para dar uma visao curta do que aconteceu sem precisar reler todos os logs.

<br><br>O resumo mostra:

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


<br><br>

![Resumo e métricas da execução](docs/images/studio-metrics.jpg#left)

As integracoes sao contabilizadas para handlers como `graphql_enrich`, `rest_call` e `notify`. Quando as integracoes reais estao desativadas, o resumo ainda registra as chamadas simuladas para deixar claro o desenho operacional do workflow.

O resumo tambem ajuda a comparar execucoes. Depois da primeira execucao, o botao **Reprocessar** fica disponivel. Ao reprocessar, o Studio usa o snapshot anterior e mostra a diferenca de tempo entre o processamento atual e o anterior.

Use esse comportamento para validar cenarios como:<br>

- uma etapa falhou e o workflow precisa continuar do ponto em que parou;
- etapas anteriores devem ser preservadas;
- uma integracao externa nao deve ser chamada de novo sem necessidade;
- o tempo total mudou depois de ajustar uma regra ou remover uma chamada.

Essa visao nao substitui metricas reais em ambiente integrado, mas ajuda muito durante a construcao do workflow. Ela deixa explicito o caminho percorrido e reduz a chance de validar apenas o resultado final sem entender como ele foi produzido.
