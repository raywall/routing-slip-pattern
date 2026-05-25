# Assert

Use `assert` quando a regra e **obrigatoria** e deve falhar o workflow se nao for atendida.

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

Tambem e possivel validar uma condicao simples:

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

Validacao de colecao:

```yaml
- name: assert
  params:
    field: itens
    min_items: 1
    message: Pedido sem itens.
```

Operadores disponiveis:

| Operador | Uso |
|---|---|
| `equals` | Igualdade. |
| `not_equals` | Diferenca. |
| `less_than` | Menor que. |
| `less_than_or_equal` | Menor ou igual. |
| `greater_than` | Maior que. |
| `greater_than_or_equal` | Maior ou igual. |
| `min_items` | Tamanho minimo de lista, mapa ou string. |
| `max_items` | Tamanho maximo de lista, mapa ou string. |
| `exists` | Existencia de path. |
