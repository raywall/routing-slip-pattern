# Configuracao e documentacao

![Configuracoes](docs/images/studio-config.jpg#right)

A aba de configuracao reune valores de teste que mudam conforme o ambiente. Ela evita alterar o YAML apenas para trocar endpoints ou definir se as integracoes serao simuladas.

| Campo | Uso |
| --- | --- |
| Exemplo | Carrega um workflow base para estudo. |
| Chamar integracoes reais | Liga chamadas reais em handlers como `graphql_enrich` e `rest_call`. |
| GraphQL endpoint | URL do `go-graphql-connector`. |
| REST workflow endpoint | Endpoint do runtime para disparo via REST. |
| External API URL | Base URL usada por exemplos e integracoes externas. |

Comece com integracoes simuladas para validar estrutura, paths, condicoes e saltos. Depois, ligue integracoes reais para validar contratos, tempo de resposta e tratamento de erro.

![Documentacao](docs/images/studio-docs.jpg#left)

A aba de documentacao fica no botao de informacao. Ela organiza o conhecimento em uma sequencia progressiva:

1. introducao;
2. construcao de workflows;
3. handlers;
4. projetos integrados;
5. uso do Studio.

O conteudo e Markdown renderizado no proprio Studio. Diagramas, tabelas, exemplos e imagens devem ser usados para tornar a leitura autonoma.

