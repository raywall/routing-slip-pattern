![Visão geral da interface do Studio](docs/images/studio-interface.jpg)

O Studio foi pensado para concentrar o trabalho de criacao, revisao e teste de workflows em uma unica tela. A ideia e evitar aquele ciclo cansativo de editar um arquivo, trocar de terminal, executar comando, procurar log e voltar para o editor sem contexto.

A interface e dividida em tres areas principais:

| Area | Para que serve |
|---|---|
| Painel lateral | Reune workspace, payload de entrada, configuracoes e documentacao. |
| Editor central | Onde o YAML do workflow e escrito, validado e salvo. |
| Area de resultado | Mostra logs, fases executadas, resumo da simulacao ou conteudo da documentacao. |

Na pratica, o fluxo de trabalho costuma ser:

1. Abrir um workspace local.
2. Escolher ou criar um workflow.
3. Ajustar o payload de entrada.
4. Executar a simulacao.
5. Ler os logs por etapa.
6. Corrigir o YAML e repetir.

Quando o workflow cresce, use o recurso de `workflow_ref` para dividir o processo em arquivos menores. O Studio consegue carregar esses arquivos pelo workspace e exportar uma versao composta quando for necessario executar tudo como um unico YAML.

O tema claro/escuro fica no topo da area principal. No celular, o Studio muda de comportamento: a experiencia vira apenas leitura da documentacao, com um menu lateral para navegar entre topicos.
