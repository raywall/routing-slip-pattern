---
sidebar_position: 2
sidebar_label: "Configuração e documentação"
---

# Configuração e documentação

![Configurações](docs/images/studio-config.jpg)

A aba de configuração reúne valores de teste que mudam conforme o ambiente. Ela evita alterar o YAML apenas para trocar endpoints ou definir se as integrações serão simuladas.

| Campo | Uso |
| --- | --- |
| Exemplo | Carrega um workflow base para estudo. |
| Chamar integrações reais | Liga chamadas reais em handlers como `graphql_enrich` e `rest_call`. |
| GraphQL endpoint | URL do `go-graphql-connector`. |
| REST workflow endpoint | Endpoint do runtime para disparo via REST. |
| MCP endpoint | Endpoint JSON-RPC usado para validar e explicar workflows pelo runtime. |
| MCP API key | Chave opcional enviada como `Authorization: Bearer` e `X-API-Key`. |
| External API URL | Base URL usada por exemplos e integrações externas. |

Comece com integrações simuladas para validar estrutura, paths, condições e saltos. Depois, ligue integrações reais para validar contratos, tempo de resposta e tratamento de erro.

## Acoes MCP

Os botoes de MCP usam o workflow aberto no editor:

| Botão | O que faz |
| --- | --- |
| Validar MCP | Envia o YAML para `validate_workflow` e retorna erros estruturais, handlers desconhecidos e saltos inválidos. |
| Explicar MCP | Envia o YAML para `explain_workflow` e mostra uma leitura resumida das etapas, controles e integrações. |
| Diagnosticar conectores | Analisa localmente as etapas `graphql_enrich`, `rest_call` e `notify`, exibindo endpoints, targets, retries e circuit breaker. |

Essas acoes nao alteram arquivos e nao executam efeitos externos. Elas servem para revisar o desenho do fluxo antes de rodar o teste.

![Documentação](docs/images/studio-docs.jpg)

A aba de documentação fica no botão de informação. Ela organiza o conhecimento em uma sequencia progressiva:

1. introdução;
2. construção de workflows;
3. handlers;
4. projetos integrados;
5. uso do Studio.

O conteúdo e Markdown renderizado no próprio Studio. Diagramas, tabelas, exemplos e imagens devem ser usados para tornar a leitura autônoma.
