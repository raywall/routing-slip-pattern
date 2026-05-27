Use `filter_array` para remover itens de um array antes das próximas etapas.

O handler pode sobrescrever o array original ou gravar a lista filtrada em outro campo.

## Filtro declarativo

```yaml
- name: filter_array
  params:
    source: catalogo.produtos
    where:
      all:
        - field: item.disponibilidade.status
          equals: DISPONIVEL
        - field: item.preco.valor
          less_than_or_equal: 100
```

## Gravar resultado em outro campo

```yaml
- name: filter_array
  params:
    source: catalogo.produtos
    target: produtos_elegiveis
    where:
      field: item.categoria
      equals: ELETRONICOS
```

## Usar CEL por item

```yaml
- name: filter_array
  params:
    source: entrega.opcoes
    target: entrega.opcoes_validas
    expr: "item.prazo_dias <= 3 && item.custo <= 25"
```

Durante a avaliação:

| Variável | Uso |
|---|---|
| `item` | Item atual do array. |
| `index` | Posição do item no array original. |
| payload original | Continua disponível para comparações. |

Campos gerados:

| Campo | Descrição |
|---|---|
| `<target>_filtered_count` | Quantidade de itens mantidos. |
| `<target>_removed_count` | Quantidade de itens removidos. |
