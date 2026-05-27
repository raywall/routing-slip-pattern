# Compute

Use `compute` para calcular e gravar um valor no payload.

```yaml
- name: compute
  params:
    target: produto_promocional
    value:
      field: catalogo.produtos.0.preco.valor
      less_than_or_equal: 100
```

Copiar valor de outro campo:

```yaml
- name: compute
  params:
    target: sku_principal
    value:
      field: itens.0.sku
```

Valor literal:

```yaml
- name: compute
  params:
    target: canal
    value:
      literal: CHECKOUT_ONLINE
```

Contagem de itens:

```yaml
- name: compute
  params:
    target: quantidade_itens
    value:
      count: itens
```

Existência de path:

```yaml
- name: compute
  params:
    target: possui_endereco
    value:
      exists: entrega.endereco
```
