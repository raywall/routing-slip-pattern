---
sidebar_position: 14
sidebar_label: "Observabilidade e AWS"
---

# Observabilidade e AWS

Alguns workflows precisam produzir sinais explícitos ou executar comandos em serviços externos. Para esses casos, o routing-slip-pattern oferece handlers específicos para log, métricas e efeitos AWS.

Use esses handlers com parcimônia e sempre em etapas bem nomeadas. A leitura e o enriquecimento continuam sendo melhor representados por `graphql_enrich`; os handlers desta seção são indicados para comandos, publicação de eventos, escrita de estado e registro de sinais operacionais.

## log

Registra um log estruturado quando existe um marco funcional ou técnico importante.

```yaml
- id: registrar-pedido-validado
  name: log
  params:
    level: info
    message: "Pedido {order_id} validado"
    fields:
      - order_id
      - customer.id
    data:
      stage: fulfillment
      source: routing-slip
    required: false
```

Campos principais:

| Campo | Descrição |
|---|---|
| `level` | `debug`, `info`, `warn` ou `error`. |
| `message` | Mensagem com interpolação por `{path}`. |
| `fields` | Paths do payload incluídos no log. |
| `data` | Objeto adicional com valores fixos ou interpolados. |
| `required` | Se `true`, falha quando um campo listado não existe. |

## datadog_metric

Envia uma métrica customizada para a API de séries do Datadog. Quando `required` é `false`, falhas no envio não interrompem o processamento.

```yaml
- id: metricar-processamento
  name: datadog_metric
  params:
    metric: routing_slip.orders.completed
    type: count
    value: 1
    tags:
      workflow: order-fulfillment
      channel: "{input.channel}"
      status: success
    api_key: "{secrets.datadog_api_key}"
    api_url: https://api.datadoghq.com/api/v1/series
    timeout_ms: 2000
    required: false
```

O handler adiciona `correlation_id:<valor>` automaticamente quando a mensagem possui correlation id. Se `api_key` não for informado no YAML, o runtime tenta usar `DATADOG_API_KEY`.

## aws_action

Executa ações controladas em serviços AWS. O mesmo handler suporta LocalStack usando `endpoint`.

Parâmetros comuns:

| Campo | Descrição |
|---|---|
| `service` | `dynamodb`, `s3`, `sqs`, `sns`, `secretsmanager` ou `ssm`. |
| `action` | Ação executada no serviço. |
| `region` | Região AWS. Padrão: `us-east-1`. |
| `endpoint` | Endpoint alternativo, como `http://localstack:4566`. |
| `target` | Path onde o resultado será salvo. |
| `required` | Controla se uma falha interrompe o workflow. |

### DynamoDB

```yaml
- id: salvar-status
  name: aws_action
  params:
    service: dynamodb
    action: put
    endpoint: http://localstack:4566
    table: workflow-items
    item:
      pk: "ORDER#{order_id}"
      sk: "STATUS"
      status: "{order.status}"
    target: dynamodb_result
```

Também é possível usar `get`, `update` e `delete`. Para `update`, informe `key`, `update_expression`, `expression_attribute_names` e `expression_attribute_values`.

### S3

```yaml
- id: gravar-payload
  name: aws_action
  params:
    service: s3
    action: put
    bucket: workflow-artifacts
    key: "orders/{order_id}/payload.json"
    body:
      order_id: "{order_id}"
      status: "{order.status}"
    target: s3_result
```

Use `get` para recuperar o arquivo em `target.body` e `delete` para remover.

### SQS e SNS

```yaml
- id: publicar-evento-fila
  name: aws_action
  params:
    service: sqs
    action: send
    queue_url: http://localstack:4566/000000000000/order-events
    message:
      type: ORDER_READY
      order_id: "{order_id}"
    target: sqs_result
```

```yaml
- id: publicar-evento-topico
  name: aws_action
  params:
    service: sns
    action: publish
    topic_arn: arn:aws:sns:us-east-1:000000000000:order-events
    subject: ORDER_READY
    message:
      order_id: "{order_id}"
      status: "{order.status}"
    target: sns_result
```

### Secrets Manager e Parameter Store

```yaml
- id: ler-chave-datadog
  name: aws_action
  params:
    service: secretsmanager
    action: get
    secret_id: /routing-slip/datadog
    target: datadog_secret
```

```yaml
- id: salvar-parametro
  name: aws_action
  params:
    service: ssm
    action: put
    name: /routing-slip/orders/max-retries
    value: "3"
    type: String
    overwrite: true
    target: parameter_result
```
