![Editor de script e atalhos](docs/images/studio-editor.jpg#right)

Atalhos disponiveis:

- Tab: indenta o trecho selecionado.
- Shift+Tab: remove indentacao.
- Cmd+/ ou Ctrl+/: comenta ou descomenta o bloco selecionado.
- Cmd+Enter ou Ctrl+Enter: executa a simulacao.
- Cmd+S ou Ctrl+S: salva o arquivo aberto no workspace.

Os logs da execucao aparecem agrupados por etapa. Clicar em um log foca a etapa correspondente no YAML.

Depois de uma execucao, o botao **Reprocessar** fica disponivel. Ele usa o snapshot da execucao anterior para retomar do cursor salvo, preservando as etapas ja processadas e executando somente o trecho restante.
