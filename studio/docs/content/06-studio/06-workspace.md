# Workspace


O workspace organiza arquivos locais. A ideia e representar contextos de negocio ou microsserviços como pastas, e workflows como arquivos YAML dentro dessas pastas.

| Elemento | Representação |
| --- | --- |
| Pasta | Microservice, domínio ou contexto. |
| Arquivo `.yaml`/`.yml` | Workflow daquele contexto. |
| Arquivo aberto | Conteúdo carregado no editor. |

![Workspace](docs/images/studio-workspace.jpg)

## Acoes disponíveis

- abrir uma pasta local;
- criar microservice/pasta;
- criar workflow;
- renomear;
- excluir;
- salvar;
- atualizar;
- exportar workflow composto.

O workspace usa a File System Access API. Chrome e Edge oferecem a melhor experiencia para abrir uma pasta com permissão de leitura e escrita.

## Organização recomendada

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

Use composição com `workflow_ref` quando um fluxo principal precisar chamar workflows de outros contextos.

