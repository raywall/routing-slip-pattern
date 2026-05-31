# Routing Slip Studio

Interface local para editar, validar e testar workflows YAML do
`routing-slip-pattern`.

## Executar

Na raiz do projeto:

```bash
make studio
```

Acesse:

```text
http://localhost:8089
```

Tambem funciona com qualquer servidor estatico apontando para esta pasta.

## Recursos

- Editor YAML para workflows.
- Sidebar redimensionavel.
- Abas laterais para Workspace, Configuracao e Payload de entrada.
- Workspace local com microservicos como pastas e workflows YAML como arquivos.
- Criacao, abertura, renomeacao, exclusao e salvamento de workflows no workspace.
- Divisor redimensionavel entre abas e editor para ajustar o espaco de trabalho.
- Indentacao com Tab e Shift+Tab no editor YAML.
- Numeros de linha sincronizados com o scroll do editor.
- Comentario/descomentario de bloco com `Cmd+/` ou `Ctrl+/`.
- Execucao do workflow pelo editor com `Cmd+Enter` ou `Ctrl+Enter`.
- Reprocessamento a partir do snapshot da execucao anterior.
- Clique em logs de etapa para focar o trecho equivalente no YAML.
- Logs agrupados por fase/step para facilitar a leitura da execucao.
- Resumo final da execucao com total de steps, erros, integracoes, tempo total e diferenca para o processamento anterior.
- Aba de documentacao navegavel, com conteudo mantido em `studio/docs`.
- Documentacao dos projetos integrados `go-graphql-connector` e `custom-business-metrics`.
- Composicao de scripts com `workflow_ref` para simular subfluxos em arquivos separados.
- Exportacao de workflow composto em um unico YAML resolvendo todos os `workflow_ref`.
- Estado salvo no IndexedDB local, com fallback para localStorage.
- Lint estrutural dos handlers suportados.
- Editor JSON para payload de entrada.
- Simulacao local da execucao passo a passo.
- Logs por etapa com payload, cursor, historico e erros.
- Envio opcional para o endpoint REST real do `routing-slip-pattern`.
- Toggle para chamadas reais de `graphql_enrich` e `rest_call`.

## Workspace

Na aba `Workspace`, clique em `Abrir` e selecione um diretorio local. Cada
pasta de primeiro nivel representa um microservico; cada arquivo `.yaml` ou
`.yml` dentro dela representa um workflow daquele contexto.

Use os botoes da aba ou o menu de contexto da arvore para criar, renomear,
excluir e salvar workflows. O atalho `Cmd+S`/`Ctrl+S` salva o workflow aberto
quando ele pertence ao workspace.

## Documentacao no Studio

A aba `Documentacao` usa os arquivos em `studio/docs` para separar o codigo e
o conteudo navegavel da documentacao. Ao clicar em um subitem, o conteudo e
exibido na area principal onde normalmente aparecem os logs da execucao.

Sempre que `DOCUMENTATION.md` for atualizado por causa de uma funcionalidade,
atualize tambem os arquivos Markdown em `studio/docs/content`. Cada arquivo usa
front matter com `sidebar_position` e `sidebar_label`; o Studio monta a arvore
de navegacao a partir de `studio/docs/manifest.json` e desses headers.

## Organizacao dos assets

Os scripts e estilos sao separados por contexto para evitar um arquivo unico
dificil de manter.

Scripts ficam em `studio/scripts` e seguem o padrao:

```text
routing-slip.studio.{contexto}.js
```

Estilos ficam em `studio/styles` e seguem o mesmo padrao:

```text
routing-slip.studio.{contexto}.css
```

Ao adicionar uma nova area da tela, crie um arquivo com o contexto correspondente
e registre a ordem de carregamento em `index.html`.

## Observacoes

O Studio valida a estrutura conhecida do workflow, mas nao substitui a execucao
real do app Go. Use a simulacao para acelerar a criacao do script e depois teste
o fluxo real com o app iniciado via:

```bash
go run ./app --config ./config.yaml --workflow ./workflows/payment-fulfillment.yaml
```
