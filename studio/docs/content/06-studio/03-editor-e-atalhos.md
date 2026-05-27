# Editor de scripts e atalhos

![Editor de script e atalhos](docs/images/studio-editor.jpg#right)

Atalhos disponíveis:

- Tab: indena o trecho selecionado.
- Shift+Tab: remove intentadão.
- Cmd+/ ou Ctrl+/: comenta ou descometa o bloco selecionado.
- Cmd+Enter ou Ctrl+Enter: executa a simulação.
- Cmd+S ou Ctrl+S: salva o arquivo aberto no workspace.

Os logs da execução aparecem agrupados por etapa. Clicar em um log foca a etapa correspondente no YAML.

Depois de uma execução, o botão **Reprocessar** fica disponível. Ele usa o snapshot da execução anterior para retomar do cursor salvo, preservando as etapas ja processadas e executando somente o trecho restante.
