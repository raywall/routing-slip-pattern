# Composicao de scripts

Workflows longos podem ficar dificeis de analisar em um unico arquivo. A composicao permite dividir um processo em arquivos menores e depois conecta-los como se fossem um fluxo unico.

Imagine um processo de venda online:

- receber pagamento;
- consultar pedido;
- preparar entrega;
- atualizar estoque;
- notificar cliente.

Cada contexto pode ficar em um arquivo proprio. O workflow principal referencia os demais com `workflow_ref`.

```yaml
name: online-sale
steps:
  - id: validate-input
    name: validate
    params:
      required:
        - correlation_id
        - order_id

  - id: delivery
    name: workflow_ref
    params:
      file: delivery/prepare-delivery.yaml
      prefix: delivery

  - id: inventory
    name: workflow_ref
    params:
      file: inventory/update-stock.yaml
      prefix: inventory
```

Ao carregar o workflow, o runtime expande as referencias. Os IDs dos steps filhos recebem prefixo, evitando conflito.

## Beneficios

- melhora leitura de fluxos extensos;
- permite organizar por dominio ou microservico;
- facilita revisao por times diferentes;
- reduz conflitos em versionamento;
- permite exportar o workflow composto para execucao.

## Decisao pratica

Use composicao quando um trecho tem responsabilidade clara e pode ser explicado sozinho. Evite dividir demais etapas pequenas, pois isso pode dificultar a navegacao.

