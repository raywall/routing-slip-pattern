# Documentacao no Studio

A aba Documentacao deve acompanhar o DOCUMENTATION.md do projeto.

Quando uma funcionalidade nova for adicionada ao routing-slip-pattern, atualize:

1. DOCUMENTATION.md, como fonte completa de conhecimento.
2. O arquivo Markdown correspondente em `studio/docs/content`.
3. `studio/docs/documentation.js`, apenas quando precisar criar, remover ou reordenar itens da arvore de navegacao.

Mantenha a ordem dos topicos evolutiva: conceitos primeiro, uso pratico depois, detalhes avancados por ultimo.

O `documentation.js` funciona como manifesto: ele define secoes, subitens, ids e o caminho do arquivo `.md`. O conteudo renderizado no Studio vem dos arquivos Markdown, o que facilita revisar exemplos, tabelas, diagramas Mermaid e textos longos sem editar strings JavaScript.
