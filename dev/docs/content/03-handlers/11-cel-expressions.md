O handler `cel` avalia uma expressao CEL e espera resultado booleano. Ele pode falhar o workflow, continuar, parar sem erro ou saltar para outra etapa quando a expressao for falsa.

O runtime disponibiliza:

| Nome | Conteudo |
|---|---|
| `payload` | Payload completo da mensagem. |
| `headers` | Headers da mensagem. |
| Variaveis de primeiro nivel | Campos do payload com nomes validos, como `pedido`, `itens`, `catalogo`. |

## Validacao obrigatoria

```yaml
- name: cel
  params:
    expr: "pedido.status == 'APROVADO' && size(itens) > 0"
    message: Pedido precisa estar aprovado e possuir itens.
    on_false: error
```

Quando `on_false` nao e informado e nao existe `to`, o comportamento padrao e `error`.

## Salto quando falso

```yaml
- id: avaliar_pedido
  name: cel
  params:
    expr: "pedido.total > 0 && entrega.endereco.cep != ''"
    on_false: jump
    to: revisar_pedido
    target: pedido_pronto_para_entrega

- name: enrich
  params:
    data:
      rota: EXPEDICAO

- id: revisar_pedido
  name: audit
  params:
    event: pedido.revisao_necessaria
    fields:
      - correlation_id
      - pedido.id
      - pedido_pronto_para_entrega
```

## Modos de on_false

| Valor | Comportamento |
|---|---|
| `error` | Falha a etapa e registra erro. |
| `fail` | Alias de `error`. |
| `jump` | Continua no step indicado em `to`. |
| `continue` | Grava o resultado e segue para a proxima etapa. |
| `stop` | Interrompe o workflow sem erro tecnico. |

## Exemplos

```yaml
- name: cel
  params:
    expr: "size(catalogo.produtos) > 0"
    message: Nenhum produto encontrado no catalogo.
```

```yaml
- name: cel
  params:
    expr: "payload.evento == 'PEDIDO_APROVADO' && payload.pedido.total >= 50"
```

```yaml
- name: cel
  params:
    expr: "entrega.tipo == 'EXPRESSA' && pedido.total >= 100"
    target: elegivel_entrega_expressa
    on_false: continue
```

O Studio simula o subconjunto mais comum de CEL: comparacoes, operadores booleanos, acesso por ponto, `size()` e `has()`. Para expressoes avancadas, valide tambem no runtime Go.
