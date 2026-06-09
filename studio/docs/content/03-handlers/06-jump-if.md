---
sidebar_position: 11
sidebar_label: "Jump_if"
---

# Jump_if

Use `jump_if` para saltar para uma etapa posterior.

```yaml
- name: jump_if
  params:
    field: produto_promocional
    equals: true
    to: finalizar
```

O destino em `to` pode ser o `id` de um step. Prefira `id`, pois handlers podem se repetir.

```yaml
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
```

Também é possível saltar pela existência de um campo:

```yaml
- name: jump_if
  params:
    exists: pagamento.autorizacao.codigo
    to: capturar_pagamento
```

Para compatibilidade, `field` com `exists: true` também é aceito:

```yaml
- name: jump_if
  params:
    field: pagamento.autorizacao.codigo
    exists: true
    to: capturar_pagamento
```

O salto deve apontar para uma etapa posterior para evitar loops acidentais.
