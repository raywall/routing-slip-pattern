# Array transform

Use `array_transform` quando o workflow precisa tratar listas com mais de uma ação: filtrar itens, alterar campos em cada item e repetir a mesma lógica em arrays internos.

Ele e útil para preparar dados retornados por APIs antes de seguir para validações, decisões ou composição com outro workflow.

## Exemplo simples

```yaml
- name: array_transform
  params:
    source: catalogo.produtos
    target: produtos_elegiveis
    filters:
      expr: "item.status == 'DISPONIVEL'"
    updates:
      - when:
          field: item.origem
          equals: MARKETPLACE
        set:
          prioridade: BAIXA
```

## Arrays aninhados

```yaml
- name: array_transform
  params:
    source: pedidos
    target: pedidos_validos
    filters:
      expr: "item.status == 'APROVADO'"
    nested:
      - source: itens
        filters:
          expr: "item.quantidade > 0"
        updates:
          - when:
              field: parent.canal
              equals: ONLINE
            set:
              origem_processamento: CHECKOUT
```

## Valores disponíveis

| Nome | Uso |
|---|---|
| `item` | Item atual do array processado. |
| `index` | Posição do item no array. |
| `parent` | Item pai quando o handler está processando um array aninhado. |
| `today` | Data atual no formato `YYYY-MM-DD`. |
| `end_of_current_month_plus_2` | Último dia do mês atual mais dois meses. |

Em `set`, use valor literal ou copie de outro campo:

```yaml
set:
  total:
    from: item.valor_original
```
