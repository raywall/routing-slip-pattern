window.RoutingSlipDocs = [
  {
    title: "Introducao",
    items: [
      {
        id: "intro-visao-geral",
        title: "Visao geral",
        content: `# Visao geral

O **Routing Slip Studio** ajuda a criar, validar e testar workflows YAML para o motor \`routing-slip-pattern\`.

O motor executa uma mensagem com payload, metadados e uma lista ordenada de etapas. Cada etapa e resolvida por um handler registrado, permitindo criar fluxos reutilizaveis e modulares sem acoplar regras de dominio ao executor.

Use o Studio para escrever o workflow, validar a estrutura, simular a execucao, testar integracoes e navegar pela documentacao sem sair da tela de trabalho.

| Conceito | Papel |
|---|---|
| **Message** | Unidade de trabalho com payload, headers e cursor. |
| **Step** | Etapa declarada no YAML. |
| **Handler** | Implementacao que executa uma etapa. |
| **StateStore** | Persistencia do cursor e snapshots para retomada. |

\`\`\`mermaid
flowchart LR
    A[Evento ou chamada REST] --> B[Routing Slip]
    B --> C[Validate]
    C --> D[Enrich]
    D --> E[Decision]
    E --> F[Audit]
    B -. metricas .-> G[Observabilidade]
\`\`\``
      },
      {
        id: "intro-objetivo",
        title: "Objetivo",
        content: `# Objetivo

A proposta do projeto e fornecer uma base resiliente, observavel e reutilizavel para workflows de qualquer dominio.

Um workflow deve conseguir:

- receber payloads diferentes;
- validar campos obrigatorios;
- enriquecer dados em fontes externas;
- tomar decisoes;
- transformar o payload;
- auditar passos importantes;
- parar com erro ou por regra funcional;
- retomar do ponto em que parou.`
      },
      {
        id: "intro-projetos",
        title: "Projetos relacionados",
        content: `# Projetos relacionados

O routing-slip-pattern se integra a dois projetos complementares:

- go-graphql-connector: camada GraphQL configuravel para integrar APIs, bases de dados, caches e outros servicos.
- custom-business-metrics: base conceitual para metricas granulares e visualizacao realtime do processamento.

A ideia e manter o motor de workflow simples e delegar integracoes e observabilidade para componentes especializados.`
      }
    ]
  },
  {
    title: "Construindo workflows",
    items: [
      {
        id: "workflow-estrutura",
        title: "Estrutura basica",
        content: `# Estrutura basica

Um workflow YAML possui metadados e uma lista de steps.

\`\`\`yaml
name: pedido-fulfillment
description: Processa pedido aprovado ate a preparacao de entrega.
error_policy: stop
message_id_path: pedido_id
correlation_id_path: correlation_id

steps:
  - name: validate
    params:
      required:
        - pedido_id
        - correlation_id
\`\`\`

Use nomes claros e paths estaveis. Isso facilita reprocessamento, metricas e suporte operacional.`
      },
      {
        id: "workflow-paths",
        title: "Paths e arrays",
        content: `# Paths e arrays

Handlers usam dot notation para acessar campos do payload.

\`\`\`yaml
pedido.itens.0.sku
catalogo.produtos.0.disponibilidade.status
entrega.endereco.cidade
\`\`\`

Indices numericos acessam arrays. Isso permite validar ou calcular valores a partir de respostas enriquecidas por GraphQL ou REST.`
      },
      {
        id: "workflow-reprocessamento",
        title: "Reprocessamento",
        content: `# Reprocessamento

O routing slip guarda o cursor da execucao. Quando uma etapa falha, o estado pode ser persistido e uma execucao futura retoma a partir da etapa que falhou.

Esse comportamento evita repetir etapas anteriores que ja produziram efeito, como emissao fiscal, envio de notificacao ou atualizacao de inventario.

Para etapas com efeito externo, combine reprocessamento com idempotencia por:

- message_id;
- step;
- attempt;
- correlation_id.`
      },
      {
        id: "workflow-composicao",
        title: "Composicao de scripts",
        content: `# Composicao de scripts

Use \`workflow_ref\` para dividir fluxos extensos em arquivos menores sem perder a execucao continua.

> Pense em cada arquivo como uma parte coesa do processo. Na execucao, o motor enxerga tudo como um unico routing slip.

\`\`\`yaml
- id: emitir_nota
  name: workflow_ref
  params:
    file: ../fiscal/emitir-nota.yaml

- id: preparar_entrega
  name: workflow_ref
  params:
    file: ../expedicao/preparar-entrega.yaml
\`\`\`

Durante a execucao, as etapas dos arquivos referenciados sao expandidas no ponto da referencia. O cursor, os logs e as metricas continuam funcionando como se fosse um unico workflow.

\`\`\`mermaid
flowchart LR
    A[pagamento-aprovado.yaml] --> B[workflow_ref fiscal]
    B --> C[emitir-nota.yaml]
    C --> D[workflow_ref expedicao]
    D --> E[preparar-entrega.yaml]
\`\`\`

| Campo | Uso |
|---|---|
| \`params.file\` | Caminho relativo para outro YAML. |
| \`params.prefix\` | Prefixo opcional para IDs expandidos. |
| \`id\` do step | Prefixo padrao quando \`params.prefix\` nao existe. |

Recomendacoes:

- mantenha cada arquivo focado em um contexto;
- use \`id\` no step \`workflow_ref\` para gerar prefixos estaveis;
- use caminhos relativos como \`../fiscal/emitir-nota.yaml\`;
- evite ciclos entre workflows;
- documente o payload esperado e produzido por cada subfluxo.`
      }
    ]
  },
  {
    title: "Handlers",
    items: [
      {
        id: "handlers-visao-geral",
        title: "Visao geral",
        content: `# Visao geral dos handlers

Handlers sao unidades pequenas e combinaveis. Cada handler recebe o payload atual, seus \`params\` e decide se altera o payload, interrompe o fluxo, registra informacoes ou chama uma integracao.

| Handler | Papel |
|---|---|
| \`validate\` | Verifica campos obrigatorios. |
| \`condition\` | Para o fluxo de forma funcional quando uma regra nao bate. |
| \`assert\` | Falha o workflow quando uma regra obrigatoria nao e atendida. |
| \`compute\` | Calcula e grava valores no payload. |
| \`cel\` | Avalia expressoes CEL e decide erro, salto, continuidade ou parada. |
| \`jump_if\` | Altera o cursor para uma etapa posterior. |
| \`enrich\` | Injeta dados estaticos no payload. |
| \`transform\` | Normaliza texto. |
| \`graphql_enrich\` | Enriquece via GraphQL Connector. |
| \`rest_call\` | Chama uma API REST e salva a resposta. |
| \`audit\` | Registra evidencia funcional. |
| \`notify\` | Simula envio de notificacao. |

O runtime suporta CEL por meio do handler \`cel\`, usando \`cel-go\`. Use validacoes declarativas para regras simples e CEL quando a expressao deixar a regra mais clara.`
      },
      {
        id: "handlers-validate",
        title: "Validate",
        content: `# Validate

Use \`validate\` para garantir que campos essenciais existem antes de executar efeitos externos.

\`\`\`yaml
- name: validate
  params:
    required:
      - pedido_id
      - correlation_id
\`\`\`

Por padrao, campos ausentes geram erro e o workflow respeita a \`error_policy\`.

\`\`\`yaml
- name: validate
  params:
    required:
      - pedido_id
      - comprador.id
      - itens.0.sku
      - entrega.endereco.cep
    stop_on_failure: true
\`\`\`

Para apenas registrar a falha e continuar:

\`\`\`yaml
- name: validate
  params:
    required:
      - metadados.origem
    stop_on_failure: false
\`\`\`

O handler grava \`validation_passed: true\` quando tudo passa e \`validation_error\` quando faltam campos.`
      },
      {
        id: "handlers-condition",
        title: "Condition",
        content: `# Condition

Use \`condition\` para interromper o fluxo de forma graciosa, sem tratar como erro tecnico.

\`\`\`yaml
- name: condition
  params:
    field: evento
    equals: PEDIDO_APROVADO
\`\`\`

Com \`not_equals\`, o fluxo para se o valor encontrado for igual ao bloqueado:

\`\`\`yaml
- name: condition
  params:
    field: pedido.status
    not_equals: CANCELADO
\`\`\`

Quando a condicao interrompe o fluxo, o payload recebe \`gate_stopped: true\`. Use esse handler para decisoes funcionais esperadas, como evento fora de escopo ou status que nao deve prosseguir.`
      },
      {
        id: "handlers-assert",
        title: "Assert",
        content: `# Assert

Use \`assert\` quando a regra e **obrigatoria** e deve falhar o workflow se nao for atendida.

\`\`\`yaml
- name: assert
  params:
    all:
      - field: catalogo.produtos.0.categoria
        equals: ELETRONICOS
      - field: catalogo.produtos.0.disponibilidade.status
        equals: DISPONIVEL
    message: Produto fora dos criterios.
\`\`\`

Tambem e possivel validar uma condicao simples:

\`\`\`yaml
- name: assert
  params:
    field: pedido.status
    equals: APROVADO
    message: Pedido precisa estar aprovado.
\`\`\`

Ou aceitar qualquer regra de uma lista:

\`\`\`yaml
- name: assert
  params:
    any:
      - field: entrega.tipo
        equals: EXPRESSA
      - field: entrega.tipo
        equals: RETIRADA
    message: Tipo de entrega nao suportado.
\`\`\`

Validacao de colecao:

\`\`\`yaml
- name: assert
  params:
    field: itens
    min_items: 1
    message: Pedido sem itens.
\`\`\`

Operadores disponiveis:

| Operador | Uso |
|---|---|
| \`equals\` | Igualdade. |
| \`not_equals\` | Diferenca. |
| \`less_than\` | Menor que. |
| \`less_than_or_equal\` | Menor ou igual. |
| \`greater_than\` | Maior que. |
| \`greater_than_or_equal\` | Maior ou igual. |
| \`min_items\` | Tamanho minimo de lista, mapa ou string. |
| \`max_items\` | Tamanho maximo de lista, mapa ou string. |
| \`exists\` | Existencia de path. |`
      },
      {
        id: "handlers-compute",
        title: "Compute",
        content: `# Compute

Use \`compute\` para calcular e gravar um valor no payload.

\`\`\`yaml
- name: compute
  params:
    target: produto_promocional
    value:
      field: catalogo.produtos.0.preco.valor
      less_than_or_equal: 100
\`\`\`

Copiar valor de outro campo:

\`\`\`yaml
- name: compute
  params:
    target: sku_principal
    value:
      field: itens.0.sku
\`\`\`

Valor literal:

\`\`\`yaml
- name: compute
  params:
    target: canal
    value:
      literal: CHECKOUT_ONLINE
\`\`\`

Contagem de itens:

\`\`\`yaml
- name: compute
  params:
    target: quantidade_itens
    value:
      count: itens
\`\`\`

Existencia de path:

\`\`\`yaml
- name: compute
  params:
    target: possui_endereco
    value:
      exists: entrega.endereco
\`\`\``
      },
      {
        id: "handlers-jump-if",
        title: "Jump_if",
        content: `# Jump_if

Use \`jump_if\` para saltar para uma etapa posterior.

\`\`\`yaml
- name: jump_if
  params:
    field: produto_promocional
    equals: true
    to: finalizar
\`\`\`

O destino em \`to\` pode ser o \`id\` de um step. Prefira \`id\`, pois handlers podem se repetir.

\`\`\`yaml
- id: avaliar_promocao
  name: compute
  params:
    target: produto_promocional
    value:
      field: catalogo.produtos.0.preco.valor
      less_than_or_equal: 100

- name: jump_if
  params:
    field: produto_promocional
    equals: true
    to: finalizar_promocao

- name: enrich
  params:
    data:
      fluxo: PADRAO

- id: finalizar_promocao
  name: audit
  params:
    event: pedido.promocional
\`\`\`

O salto deve apontar para uma etapa posterior para evitar loops acidentais.`
      },
      {
        id: "handlers-enrich-transform",
        title: "Enrich e transform",
        content: `# Enrich e transform

Use \`enrich\` para adicionar dados ao payload.

\`\`\`yaml
- name: enrich
  params:
    data:
      origem: CHECKOUT_ONLINE
      prioridade: NORMAL
\`\`\`

Com prefixo:

\`\`\`yaml
- name: enrich
  params:
    prefix: meta_
    data:
      origem: CHECKOUT_ONLINE
\`\`\`

Use \`transform\` para normalizar texto.

\`\`\`yaml
- name: transform
  params:
    field: comprador.email
    operation: lowercase
    target: comprador_email_normalizado
\`\`\`

Operacoes suportadas: \`uppercase\`, \`lowercase\`, \`trim\`, \`prefix:<valor>\` e \`suffix:<valor>\`.`
      },
      {
        id: "handlers-graphql",
        title: "GraphQL enrich",
        content: `# GraphQL enrich

Use \`graphql_enrich\` para buscar dados no go-graphql-connector.

\`\`\`yaml
- name: graphql_enrich
  params:
    query: "query ($pedidoID: String!) { dataSources(pedidoID: $pedidoID) { order { pedido_id status } } }"
    variables:
      pedidoID: "{pedido_id}"
    target: pedido
    result_path: dataSources.order
    required: true
\`\`\`

Com variaveis interpoladas:

\`\`\`yaml
- name: graphql_enrich
  params:
    endpoint: "\${GRAPHQL_ENDPOINT:-http://localhost:8090/graphql}"
    query: "query ($sku: String!) { dataSources(sku: $sku) { catalogo { produtos { sku preco { valor } } } } }"
    variables:
      sku: "{itens.0.sku}"
    target: catalogo
    result_path: dataSources.catalogo
    timeout_ms: 3000
    required: true
\`\`\`

Se \`required: false\`, falhas de endpoint marcam \`<target>_partial: true\` e permitem continuar.`
      },
      {
        id: "handlers-rest",
        title: "Rest call",
        content: `# Rest call

Use \`rest_call\` quando o workflow precisa acionar uma API REST diretamente.

\`\`\`yaml
- name: rest_call
  params:
    base_url: "https://mock.raysouz.studio"
    method: POST
    endpoint: /expedicao
    target: expedicao
\`\`\`

Exemplo com POST, body e headers:

\`\`\`yaml
- name: rest_call
  params:
    base_url: "https://mock.raysouz.studio"
    method: POST
    endpoint: /entregas
    target: entrega
    headers:
      x-correlation-id: "{correlation_id}"
    body:
      pedido_id: "{pedido_id}"
      itens: "{itens}"
    result_path: data
    timeout_ms: 3000
    required: true
\`\`\`

Assim como no GraphQL, \`required: false\` permite continuar e marca resposta parcial.`
      },
      {
        id: "handlers-audit-notify",
        title: "Audit e notify",
        content: `# Audit e notify

Use \`audit\` para registrar evidencia funcional.

\`\`\`yaml
- name: audit
  params:
    event: pedido.processado
    fields:
      - pedido_id
      - correlation_id
      - entrega.status
\`\`\`

Use \`notify\` para simular uma notificacao.

\`\`\`yaml
- name: notify
  params:
    channel: webhook
    recipient: "https://example.local/hook"
    template: "Pedido {pedido_id} processado com status {entrega.status}"
\`\`\`

Em producao, \`notify\` pode receber uma funcao de envio real no registro do handler.`
      },
      {
        id: "handlers-cel",
        title: "CEL expressions",
        content: `# CEL expressions

O handler \`cel\` avalia uma expressao CEL e espera resultado booleano. Ele pode falhar o workflow, continuar, parar sem erro ou saltar para outra etapa quando a expressao for falsa.

O runtime disponibiliza:

| Nome | Conteudo |
|---|---|
| \`payload\` | Payload completo da mensagem. |
| \`headers\` | Headers da mensagem. |
| Variaveis de primeiro nivel | Campos do payload com nomes validos, como \`pedido\`, \`itens\`, \`catalogo\`. |

## Validacao obrigatoria

\`\`\`yaml
- name: cel
  params:
    expr: "pedido.status == 'APROVADO' && size(itens) > 0"
    message: Pedido precisa estar aprovado e possuir itens.
    on_false: error
\`\`\`

Quando \`on_false\` nao e informado e nao existe \`to\`, o comportamento padrao e \`error\`.

## Salto quando falso

\`\`\`yaml
- id: avaliar_pedido
  name: cel
  params:
    expr: "pedido.total > 0 && entrega.endereco.cep != ''"
    on_false: jump
    to: revisar_pedido
    target: pedido_pronto_para_entrega

- name: enrich
  params:
    data:
      rota: EXPEDICAO

- id: revisar_pedido
  name: audit
  params:
    event: pedido.revisao_necessaria
    fields:
      - correlation_id
      - pedido.id
      - pedido_pronto_para_entrega
\`\`\`

## Modos de on_false

| Valor | Comportamento |
|---|---|
| \`error\` | Falha a etapa e registra erro. |
| \`fail\` | Alias de \`error\`. |
| \`jump\` | Continua no step indicado em \`to\`. |
| \`continue\` | Grava o resultado e segue para a proxima etapa. |
| \`stop\` | Interrompe o workflow sem erro tecnico. |

## Exemplos

\`\`\`yaml
- name: cel
  params:
    expr: "size(catalogo.produtos) > 0"
    message: Nenhum produto encontrado no catalogo.
\`\`\`

\`\`\`yaml
- name: cel
  params:
    expr: "payload.evento == 'PEDIDO_APROVADO' && payload.pedido.total >= 50"
\`\`\`

\`\`\`yaml
- name: cel
  params:
    expr: "entrega.tipo == 'EXPRESSA' && pedido.total >= 100"
    target: elegivel_entrega_expressa
    on_false: continue
\`\`\`

O Studio simula o subconjunto mais comum de CEL: comparacoes, operadores booleanos, acesso por ponto, \`size()\` e \`has()\`. Para expressoes avancadas, valide tambem no runtime Go.`
      }
    ]
  },
  {
    title: "Potencial",
    items: [
      {
        id: "potencial-beneficios",
        title: "Beneficios",
        content: `# Beneficios e potencial

O routing-slip-pattern combina **velocidade de construcao**, **clareza operacional** e **capacidade de reprocessamento**.

| Beneficio | Impacto |
|---|---|
| Velocidade | Workflows sao declarados em YAML, sem recompilar o motor. |
| Facilidade | Handlers pequenos reduzem complexidade cognitiva. |
| Rastreabilidade | Cada etapa registra historico, cursor e erros. |
| Observabilidade | Eventos e metricas permitem dashboards realtime. |
| Explicabilidade | O YAML mostra o motivo de cada decisao. |
| Transparencia | Logs por fase ajudam a entender entrada, regra e saida. |
| Reprocessamento | Execucoes retomam do ponto de falha. |
| Modularidade | \`workflow_ref\` divide fluxos extensos em scripts menores. |
| Escalabilidade | Integracoes e metricas podem evoluir separadamente. |

\`\`\`mermaid
flowchart LR
    A[Ideia de processo] --> B[YAML modular]
    B --> C[Lint e simulacao]
    C --> D[Execucao observavel]
    D --> E[Metricas realtime]
    D --> F[Reprocessamento granular]
    B --> G[Reuso por workflow_ref]
\`\`\`

Na pratica, a ferramenta ajuda equipes a transformar processos complexos em fluxos compreensiveis, testaveis e auditaveis.`
      }
    ]
  },
  {
    title: "Studio",
    items: [
      {
        id: "studio-workspace",
        title: "Workspace",
        content: `# Workspace

A aba Workspace organiza scripts por contexto.

- Pastas representam microservicos.
- Arquivos \`.yaml\` ou \`.yml\` representam workflows.
- Clicar em um arquivo carrega o workflow no editor.
- Cmd+S ou Ctrl+S salva o workflow aberto.
- O botao de exportacao gera um YAML unico com todos os \`workflow_ref\` resolvidos.

O workspace usa a File System Access API. Use Chrome ou Edge para abrir uma pasta local com permissao de leitura e escrita.`
      },
      {
        id: "studio-editor",
        title: "Editor e atalhos",
        content: `# Editor e atalhos

Atalhos disponiveis:

- Tab: indenta o trecho selecionado.
- Shift+Tab: remove indentacao.
- Cmd+/ ou Ctrl+/: comenta ou descomenta o bloco selecionado.
- Cmd+Enter ou Ctrl+Enter: executa a simulacao.
- Cmd+S ou Ctrl+S: salva o arquivo aberto no workspace.

Os logs da execucao aparecem por fase. Clicar em um log foca a etapa correspondente no YAML.`
      },
      {
        id: "studio-documentacao",
        title: "Documentacao no Studio",
        content: `# Documentacao no Studio

A aba Documentacao deve acompanhar o DOCUMENTATION.md do projeto.

Quando uma funcionalidade nova for adicionada ao routing-slip-pattern, atualize:

1. DOCUMENTATION.md, como fonte completa de conhecimento.
2. studio/docs/documentation.js, como versao navegavel dentro do Studio.

Mantenha a ordem dos topicos evolutiva: conceitos primeiro, uso pratico depois, detalhes avancados por ultimo.`
      }
    ]
  }
];
