---
sidebar_position: 6
sidebar_label: "Workspace"
---

# Workspace


O workspace organiza arquivos locais. A ideia e representar contextos de negocio ou microsserviços como pastas, e workflows como arquivos YAML dentro dessas pastas.

| Elemento | Representação |
| --- | --- |
| Pasta | Microservice, domínio ou contexto. |
| Arquivo `.yaml`/`.yml` | Usecase daquele contexto. |
| Subitem do arquivo | Regra de negócio associada ao usecase. |
| Arquivo aberto | Conteúdo carregado no editor. |

![Workspace](docs/images/studio-workspace.jpg)

## Acoes disponíveis

- abrir uma pasta local;
- criar microservice/pasta;
- criar workflow;
- criar regra de negócio;
- visualizar, editar e excluir regras de negócio;
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

Referencie workflows pelo path a partir da raiz do workspace:

```yaml
- id: delivery
  name: workflow_ref
  params:
    file: delivery/prepare-delivery
    prefix: delivery
```

Esse formato funciona independentemente da pasta do arquivo aberto. O lint do Studio valida se `delivery/prepare-delivery.yaml` ou `delivery/prepare-delivery.yml` existe no workspace.

## Arquivo de projeto

O Studio consegue abrir workflows YAML puros. Quando você salva pelo workspace, o arquivo também pode guardar o contexto de projeto usado durante os testes:

- configuração de endpoints e integrações;
- payload de entrada;
- script do workflow;
- regras de negócio do usecase.

Esse formato permite fechar o navegador, abrir o workspace novamente e continuar com o mesmo ambiente de validação.

```yaml
service: examples
usecase: receive-order

project_settings:
  use_real_integrations: true
  integrations:
    graphql_endpoint: http://localhost:8090/graphql
    rest_workflow_endpoint: http://localhost:8088/process
    external_api_url: https://mock.example.test
  mcp_server:
    mcp_endpoint: http://localhost:9091/mcp
    mcp_api_key: ""

payload_data: |
  {
    "request_id": "REQ-1001",
    "correlation_id": "7331809a-1b6a-4636-9b76-c5b4f483136b"
  }

workflow_script:
  name: receive-order
  error_policy: stop
  message_id_path: request_id
  correlation_id_path: correlation_id
  steps:
    - name: validate
      params:
        required:
          - request_id

business_rules: []
```

Ao exportar um workflow composto, apenas o `workflow_script` é usado. As configurações, payload e regras continuam sendo informações do Studio.

Os comentários dentro de `workflow_script` fazem parte do projeto. Eles são carregados no editor e salvos novamente junto com o script, pois normalmente registram origem da regra, decisões de implementação e observações úteis para revisão.

## Regras de negócio

Regras de negócio documentam as decisões do usecase. Elas ajudam pessoas a entenderem o motivo da decisão, ajudam engenharia a localizar a implementação e ajudam IA/MCP a interpretar o fluxo com mais contexto.

Uma regra possui:

- `human_context`: explicação em linguagem de negócio;
- `engineering_context`: aplicação, tipo, repositório e entrypoint;
- `ai_logic`: orientação para análise por LLM/MCP;
- `technical_metadata`: dependências, monitores, métricas e marcadores de log.

No workspace, as regras aparecem abaixo do usecase. Clique na regra para abrir um formulário de visualização com as visões humana, engenharia, IA, observabilidade e dependências. O Studio não expõe o YAML cru nesta tela: isso reduz erro de digitação, preserva a estrutura esperada e deixa a edição mais segura.

Use **Editar** para liberar os campos do formulário. Quando existir mais de uma regra no usecase, os botões **Anterior** e **Próxima** permitem navegar sem voltar para a árvore do workspace. Se a regra depender de outra regra do mesmo usecase, a dependência aparece como link; ao abrir a regra dependida, o botão **Voltar** retorna para a regra de origem.

Ao excluir uma regra, o Studio alerta quando outra regra do mesmo usecase declara dependência dela.
