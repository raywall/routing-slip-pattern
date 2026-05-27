![Configurações](docs/images/studio-config.jpg#right)

A aba **Configuracao** concentra os valores que mudam conforme o ambiente de teste. Ela evita que voce precise alterar o YAML toda vez que quiser apontar para outro endpoint ou decidir se as integracoes devem ser reais ou simuladas.

Campos mais usados:

| Campo | Uso |
|---|---|
| Exemplo | Carrega um workflow inicial para estudo ou teste rapido. |
| Chamar integracoes reais | Quando ligado, handlers como `graphql_enrich` e `rest_call` tentam chamar os endpoints configurados. |
| GraphQL endpoint | URL do `go-graphql-connector`. |
| REST workflow endpoint | URL usada para enviar o payload ao runtime via REST. |
| External API URL | Base URL usada por exemplos e simulacoes de chamadas externas. |

Durante a criacao do workflow, comece com integracoes simuladas. Isso ajuda a validar estrutura, paths, condicoes e saltos sem depender de servicos externos. Depois, ligue as integracoes reais para testar contratos, timeouts e respostas de APIs.

O botao **Carregar** troca o exemplo aberto no editor. O botao **Enviar REST** usa o endpoint configurado para disparar o workflow fora da simulacao local do Studio.

![Documentações](docs/images/studio-docs.jpg#left)

A aba **Documentacao** funciona como referencia de trabalho. Ela foi incluida no Studio para evitar que o usuario precise alternar entre editor, README, documentacao externa e codigo-fonte enquanto esta montando um workflow.

Os topicos sao organizados em uma sequencia progressiva:

1. conceitos do projeto;
2. construcao de workflows;
3. handlers;
4. projetos integrados;
5. beneficios e potencial;
6. uso do Studio.

Ao clicar em um subtopico, o conteudo aparece na area principal. O Studio lembra o ultimo topico lido, entao se a pagina for recarregada voce volta para o mesmo ponto da documentacao.

No desktop, a documentacao convive com o ambiente de desenvolvimento. No celular, o Studio entra em modo leitura: o editor e os controles de teste somem, e a navegacao passa a ser feita por um menu lateral dedicado aos topicos.

Sempre que uma funcionalidade nova for adicionada, atualize o Markdown correspondente em `studio/docs/content`. O arquivo `studio/docs/documentation.js` deve ser alterado apenas quando for necessario criar, remover ou reordenar topicos da arvore.
