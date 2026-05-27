# Logs e resumo de uma execução

![Logs de execução das etapas](docs/images/studio-result.jpg#right-table)

A area de resultado mostra a execução do workflow em ordem cronológica. Cada grupo representa um trecho do processamento, normalmente associado a uma etapa do YAML.

Dentro de um grupo voce pode encontrar logs de validação, enriquecimento, simulação de integração, decisão, erro ou conclusão. A leitura por etapa ajuda a responder perguntas bem praticas:<br>

- qual etapa foi executada;
- qual entrada foi usada;
- qual regra foi aplicada;
- qual Saida foi produzida;
- onde o processamento parou;
- o que precisa ser corrigido antes de reexecutar.

Quando um log corresponde a uma etapa do YAML, clicar nele leva o foco para o trecho equivalente no editor. Isso e especialmente util em scripts longos, quando procurar manualmente pela etapa quebraria o ritmo da analise.

Ao final de cada simulação, o Studio adiciona um resumo no fim da timeline. Ele serve para dar uma visão curta do que aconteceu sem precisar reler todos os logs.

<br><br>O resumo mostra:

| Indicador | Descrição |
|---|---|
| Steps executados | Quantidade de etapas processadas. |
| Steps preservados | Etapas ja processadas antes do reprocessamento. |
| Steps pulados/parados | Etapas que encerraram ou desviaram o fluxo. |
| Erros | Falhas registradas durante a simulação. |
| Tempo médio por step | Media de duração das etapas executadas. |
| Trace ID | Identificador técnico usado para correlacionar logs, métricas e chamadas externas. |
| Correlation ID | Identificador funcional do processo de negocio. |
| Integrações API/serviço | Total de chamadas ou simulações de integração. |
| Tempo total | Duração completa da simulação. |
| Dif. tempo anterior | Diferença entre o tempo do processamento atual e o anterior, quando for reprocessamento. |


<br><br>

![Resumo e métricas da execução](docs/images/studio-metrics.jpg#left)

As integrações sao contabilizadas para handlers como `graphql_enrich`, `rest_call` e `notify`. Quando as integrações reais estão desativadas, o resumo ainda registra as chamadas simuladas para deixar claro o desenho operacional do workflow. Cada integração registrada no resumo carrega o `trace_id`, permitindo relacionar a linha do resumo com logs, metrics e chamadas externas.

O resumo também ajuda a comparar execuções. Depois da primeira execução, o botão **Reprocessar** fica disponível. Ao reprocessar, o Studio usa o snapshot anterior e mostra a diferença de tempo entre o processamento atual e o anterior.

Use esse comportamento para validar cenários como:<br>

- uma etapa falhou e o workflow precisa continuar do ponto em que parou;
- etapas anteriores devem ser preservadas;
- uma integração externa nao deve ser chamada de novo sem necessidade;
- o tempo total mudou depois de ajustar uma regra ou remover uma chamada.

Essa visão nao substitui métricas reais em ambiente integrado, mas ajuda muito durante a construção do workflow. Ela deixa explicito o caminho percorrido e reduz a chance de validar apenas o resultado final sem entender como ele foi produzido.
