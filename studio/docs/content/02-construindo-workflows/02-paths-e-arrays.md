Handlers usam dot notation para acessar campos do payload.

```yaml
pedido.itens.0.sku
catalogo.produtos.0.disponibilidade.status
entrega.endereco.cidade
```

Indices numericos acessam arrays. Isso permite validar ou calcular valores a partir de respostas enriquecidas por GraphQL ou REST.
