const examples = {
  payment: {
    workflow: `name: payment-event-fulfillment
description: Evento de pagamento aprovado consulta pedido, emite nota fiscal, aciona expedicao e baixa estoque.
error_policy: stop
message_id_path: payload.pagamento_id
correlation_id_path: correlation_id
steps:
  - name: validate
    params:
      required:
        - evento
        - payload.pagamento_id
        - payload.pedido_id
        - payload.valor_pago

  - name: condition
    params:
      field: evento
      equals: PAGAMENTO_APROVADO

  - name: graphql_enrich
    params:
      query: "query ($pedidoID: String!) { dataSources(pedidoID: $pedidoID) { order { pedido_id cliente_id status valor_total endereco_entrega itens { produto_id quantidade } } } }"
      variables:
        pedidoID: "{payload.pedido_id}"
      target: pedido
      result_path: dataSources.order
      timeout_ms: 1500
      required: true

  - name: rest_call
    params:
      base_url: "\${EXTERNAL_API_URL:-https://mock.raysouz.studio}"
      method: POST
      endpoint: /lambda/notas-fiscais
      target: nota_fiscal
      body:
        pedido_id: "{pedido.pedido_id}"
        cliente_id: "{pedido.cliente_id}"
        valor_total: "{pedido.valor_total}"
        itens: "{pedido.itens}"
        pagamento_id: "{payload.pagamento_id}"
      required: true

  - name: audit
    params:
      event: payment.fulfillment.completed
      fields:
        - correlation_id
        - payload.pedido_id
        - pedido.status
        - nota_fiscal.status
`,
    payload: {
      evento: "PAGAMENTO_APROVADO",
      payload: {
        pagamento_id: "PAG-5544",
        pedido_id: "PED-9988",
        valor_pago: 150,
      },
      correlation_id: "corr-payment-fulfillment-001",
      received_at: "2026-05-21T00:00:00Z",
    },
  },
  baixa: {
    workflow: `name: Processamento de desconto em folha - baixa de parcelas
description: Preparacao da baixa de parcelas ate antes da execucao do Step Functions.
error_policy: stop
message_id_path: data.codigo_identificador_evento
correlation_id_path: correlation_id
steps:
  - name: validate
    params:
      required:
        - data.codigo_identificador_evento
        - data.event_name
        - data.codigo_identificacao_pessoa
        - data.codigo_identificacao_operacao_credito
        - data.valor_desconto

  - name: condition
    params:
      field: data.event_name
      equals: DESCONTO_FOLHA_REALIZADO

  - name: graphql_enrich
    params:
      query: "query ($codigoCliente: String!, $identificadorOperacaoCredito: String!, $dataPosicaoCalculo: String!) { dataSources(codigoCliente: $codigoCliente, identificadorOperacaoCredito: $identificadorOperacaoCredito, dataPosicaoCalculo: $dataPosicaoCalculo) { custodias { operacaoId situacaoOperacao siglaCustodia saldoDevedor } saldos { saldo { saldo_liquido_operacao } } } }"
      variables:
        codigoCliente: "{data.codigo_identificacao_pessoa}"
        identificadorOperacaoCredito: "{data.codigo_identificacao_operacao_credito}"
        dataPosicaoCalculo: "2025-05-13"
      target: baixa_contexto
      result_path: dataSources
      timeout_ms: 3000
      required: true

  - name: enrich
    params:
      data:
        workflow_input_preparado:
          status: PRONTO_PARA_STEP_FUNCTIONS
          codigo_origem_desconto: DESCONTO_EM_FOLHA

  - name: audit
    params:
      event: baixa_parcelas.preparacao.completed
      fields:
        - correlation_id
        - data.codigo_identificador_evento
        - workflow_input_preparado.status
`,
    payload: {
      correlation_id: "corr-baixa-parcelas-001",
      received_at: "2026-05-21T00:00:00Z",
      data: {
        event_name: "DESCONTO_FOLHA_REALIZADO",
        codigo_identificador_evento: "evt-abc12345-def6-7890-ghij-klmn12345678",
        codigo_identificacao_pessoa: "12345678901",
        codigo_identificacao_operacao_credito: "2699999999",
        valor_desconto: 82.6,
      },
    },
  },
  blank: {
    workflow: `name: novo-workflow
description: Workflow em construcao.
error_policy: stop
message_id_path: id
correlation_id_path: correlation_id
steps:
  - name: validate
    params:
      required:
        - id
        - correlation_id

  - name: audit
    params:
      event: novo_workflow.completed
      fields:
        - id
        - correlation_id
`,
    payload: {
      id: "MSG-001",
      correlation_id: "corr-001",
    },
  },
};
