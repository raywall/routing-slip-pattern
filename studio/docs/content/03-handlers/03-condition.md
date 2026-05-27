# Condition

Use `condition` para interromper o fluxo de forma graciosa, sem tratar como erro técnico.

```yaml
- name: condition
  params:
    field: evento
    equals: PEDIDO_APROVADO
```

Com `not_equals`, o fluxo para se o valor encontrado for igual ao bloqueado:

```yaml
- name: condition
  params:
    field: pedido.status
    not_equals: CANCELADO
```

Quando a condição interrompe o fluxo, o payload recebe `gate_stopped: true`. Use esse handler para decisões funcionais esperadas, como evento fora de escopo ou status que nao deve prosseguir.
