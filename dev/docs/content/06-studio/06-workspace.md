# Workspace

![Workspace](docs/images/studio-workspace.jpg#right)

O workspace organiza arquivos locais. A ideia e representar contextos de negocio ou microservicos como pastas, e workflows como arquivos YAML dentro dessas pastas.

| Elemento | Representacao |
| --- | --- |
| Pasta | Microservico, dominio ou contexto. |
| Arquivo `.yaml`/`.yml` | Workflow daquele contexto. |
| Arquivo aberto | Conteudo carregado no editor. |

## Acoes disponiveis

- abrir uma pasta local;
- criar microservico/pasta;
- criar workflow;
- renomear;
- excluir;
- salvar;
- atualizar;
- exportar workflow composto.

O workspace usa a File System Access API. Chrome e Edge oferecem a melhor experiencia para abrir uma pasta com permissao de leitura e escrita.

## Organizacao recomendada

```text
workflows/
  orders/
    order-processing.yaml
    load-order.yaml
  delivery/
    prepare-delivery.yaml
  inventory/
    update-stock.yaml
```

Use composicao com `workflow_ref` quando um fluxo principal precisar chamar workflows de outros contextos.

