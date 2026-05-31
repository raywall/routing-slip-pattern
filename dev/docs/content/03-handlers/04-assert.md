---
sidebar_position: 2
sidebar_label: "Assert"
---

# Assert

Use `assert` quando a regra e **obrigatória** e deve falhar o workflow se nao for atendida.

```yaml
- name: assert
  params:
    all:
      - field: catalogo.produtos.0.categoria
        equals: ELETRONICOS
      - field: catalogo.produtos.0.disponibilidade.status
        equals: DISPONIVEL
    message: Produto fora dos criterios.
```

Também e possível validar uma condição simples:

```yaml
- name: assert
  params:
    field: pedido.status
    equals: APROVADO
    message: Pedido precisa estar aprovado.
```

Ou aceitar qualquer regra de uma lista:

```yaml
- name: assert
  params:
    any:
      - field: entrega.tipo
        equals: EXPRESSA
      - field: entrega.tipo
        equals: RETIRADA
    message: Tipo de entrega nao suportado.
```

Validação de coleção:

```yaml
- name: assert
  params:
    field: itens
    min_items: 1
    message: Pedido sem itens.
```

Operadores disponíveis:

| Operador | Uso |
|---|---|
| `equals` | Igualdade. |
| `not_equals` | Diferença. |
| `less_than` | Menor que. |
| `less_than_or_equal` | Menor ou igual. |
| `greater_than` | Maior que. |
| `greater_than_or_equal` | Maior ou igual. |
| `min_items` | Tamanho mínimo de lista, mapa ou string. |
| `max_items` | Tamanho máximo de lista, mapa ou string. |
| `exists` | Existência de path. |
