---
sidebar_position: 5
sidebar_label: "Resumo detalhado da execução"
---

# Logs e resumo de uma execução

![Logs de execução das etapas](docs/images/studio-result.jpg#right-table)

A area de resultado mostra a execução do workflow em ordem cronológica. Cada grupo representa um trecho do processamento, normalmente associado a uma etapa do YAML.

Quando o workflow usa composição com `workflow_ref`, os logs passam a ser separados por abas. Cada aba representa um arquivo executado: o workflow principal aparece em uma aba própria e cada workflow referenciado ganha sua própria visão. Isso evita misturar etapas de arquivos diferentes em uma timeline única.

Cada aba possui seu próprio resumo. No workflow principal, o tempo total representa a execução completa, incluindo os workflows chamados por composição. Nas abas dos workflows referenciados, o resumo considera apenas as etapas daquele arquivo.

Dentro de um grupo voce pode encontrar logs de validação, enriquecimento, simulação de integração, decisão, erro ou conclusão. A leitura por etapa ajuda a responder perguntas bem praticas:<br>

- qual etapa foi executada;
- qual entrada foi usada;
- qual regra foi aplicada;
- qual Saida foi produzida;
- onde o processamento parou;
- o que precisa ser corrigido antes de reexecutar.

Quando um log corresponde a uma etapa do YAML, clicar nele leva o foco para o trecho equivalente no editor. Em workflows compostos, o Studio abre o arquivo de origem do log e seleciona a etapa dentro dele. Isso e especialmente util em scripts longos, quando procurar manualmente pela etapa quebraria o ritmo da analise.

Durante a execução, a área de logs fica coberta por uma camada translúcida de processamento. Ela bloqueia cliques nas abas, nos grupos de etapa e na timeline enquanto o workflow ainda está gerando eventos. Quando o processamento termina, a camada desaparece e os logs voltam a ficar navegáveis.

Ao final de cada simulação, o Studio adiciona um resumo no fim da timeline. Ele serve para dar uma visão curta do que aconteceu sem precisar reler todos os logs.

<br><br>O resumo mostra:

| Indicador | Descrição |
|---|---|
| Steps executados | Quantidade de etapas processadas. |
| Steps preservados | Etapas ja processadas antes do reprocessamento. |
| Steps pulados/parados | Etapas que encerraram ou desviaram o fluxo. |
| Erros | Falhas registradas durante a simulação. |
| Tempo médio por step | Media de duração das etapas executadas. |
| Integrações API/serviço | Total de chamadas ou simulações de integração. |
| Tempo total | Duração completa da simulação. |
| Dif. tempo anterior | Diferença entre o tempo do processamento atual e o anterior, quando for reprocessamento. |

O `trace_id` e o `correlation_id` aparecem abaixo do grid de métricas em campos de leitura. Eles ficam separados dos indicadores numéricos porque normalmente são textos longos e precisam de mais espaço para leitura e cópia. O `trace_id` permite correlacionar logs, métricas e chamadas externas; o `correlation_id` identifica o processo de negócio.


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
