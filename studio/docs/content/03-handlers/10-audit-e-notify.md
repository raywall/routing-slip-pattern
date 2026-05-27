# Audit e Notify

Use `audit` para registrar evidencia funcional.

```yaml
- name: audit
  params:
    event: pedido.processado
    fields:
      - pedido_id
      - correlation_id
      - entrega.status
```

Use `notify` para simular uma notificação.

```yaml
- name: notify
  params:
    channel: webhook
    recipient: "https://example.local/hook"
    template: "Pedido {pedido_id} processado com status {entrega.status}"
```

Em produção, `notify` pode receber uma função de envio real no registro do handler.
