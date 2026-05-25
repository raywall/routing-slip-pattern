# Estrutura basica

Um workflow YAML possui metadados e uma lista de steps.

```yaml
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
```

Use nomes claros e paths estaveis. Isso facilita reprocessamento, metricas e suporte operacional.
