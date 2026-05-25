# Composicao de scripts

Use `workflow_ref` para dividir fluxos extensos em arquivos menores sem perder a execucao continua.

> Pense em cada arquivo como uma parte coesa do processo. Na execucao, o motor enxerga tudo como um unico routing slip.

```yaml
- id: emitir_nota
  name: workflow_ref
  params:
    file: ../fiscal/emitir-nota.yaml

- id: preparar_entrega
  name: workflow_ref
  params:
    file: ../expedicao/preparar-entrega.yaml
```

Durante a execucao, as etapas dos arquivos referenciados sao expandidas no ponto da referencia. O cursor, os logs e as metricas continuam funcionando como se fosse um unico workflow.

```mermaid
flowchart LR
    A[pagamento-aprovado.yaml] --> B[workflow_ref fiscal]
    B --> C[emitir-nota.yaml]
    C --> D[workflow_ref expedicao]
    D --> E[preparar-entrega.yaml]
```

| Campo | Uso |
|---|---|
| `params.file` | Caminho relativo para outro YAML. |
| `params.prefix` | Prefixo opcional para IDs expandidos. |
| `id` do step | Prefixo padrao quando `params.prefix` nao existe. |

Recomendacoes:

- mantenha cada arquivo focado em um contexto;
- use `id` no step `workflow_ref` para gerar prefixos estaveis;
- use caminhos relativos como `../fiscal/emitir-nota.yaml`;
- evite ciclos entre workflows;
- documente o payload esperado e produzido por cada subfluxo.
