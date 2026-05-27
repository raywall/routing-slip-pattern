# Configuracao e documentacao

![Configuracoes](docs/images/studio-config.jpg#right)

A aba de configuracao reune valores de teste que mudam conforme o ambiente. Ela evita alterar o YAML apenas para trocar endpoints ou definir se as integracoes serao simuladas.

| Campo | Uso |
| --- | --- |
| Exemplo | Carrega um workflow base para estudo. |
| Chamar integracoes reais | Liga chamadas reais em handlers como `graphql_enrich` e `rest_call`. |
| GraphQL endpoint | URL do `go-graphql-connector`. |
| REST workflow endpoint | Endpoint do runtime para disparo via REST. |
| MCP endpoint | Endpoint JSON-RPC usado para validar e explicar workflows pelo runtime. |
| MCP API key | Chave opcional enviada como `Authorization: Bearer` e `X-API-Key`. |
| External API URL | Base URL usada por exemplos e integracoes externas. |

Comece com integracoes simuladas para validar estrutura, paths, condicoes e saltos. Depois, ligue integracoes reais para validar contratos, tempo de resposta e tratamento de erro.

## Acoes MCP

Os botoes de MCP usam o workflow aberto no editor:

| Botao | O que faz |
| --- | --- |
| Validar MCP | Envia o YAML para `validate_workflow` e retorna erros estruturais, handlers desconhecidos e saltos invalidos. |
| Explicar MCP | Envia o YAML para `explain_workflow` e mostra uma leitura resumida das etapas, controles e integracoes. |
| Diagnosticar conectores | Analisa localmente as etapas `graphql_enrich`, `rest_call` e `notify`, exibindo endpoints, targets, retries e circuit breaker. |

Essas acoes nao alteram arquivos e nao executam efeitos externos. Elas servem para revisar o desenho do fluxo antes de rodar o teste.

![Documentacao](docs/images/studio-docs.jpg#left)

A aba de documentacao fica no botao de informacao. Ela organiza o conhecimento em uma sequencia progressiva:

1. introducao;
2. construcao de workflows;
3. handlers;
4. projetos integrados;
5. uso do Studio.

O conteudo e Markdown renderizado no proprio Studio. Diagramas, tabelas, exemplos e imagens devem ser usados para tornar a leitura autonoma.
