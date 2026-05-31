# Editor de scripts e atalhos

![Editor de script e atalhos](docs/images/studio-editor.jpg#right)

Atalhos disponíveis:

- Tab: indena o trecho selecionado.
- Shift+Tab: remove intentadão.
- Cmd+/ ou Ctrl+/: comenta ou descometa o bloco selecionado.
- Cmd+Enter ou Ctrl+Enter: processa a simulação.
- Cmd+S ou Ctrl+S: salva o arquivo aberto no workspace.

Os logs da execução aparecem agrupados por etapa. Clicar em um log foca a etapa correspondente no YAML.

O botão **Diagrama** abre uma visualização navegável do workflow. A visão **Macro** mostra a composição geral, conectores e integrações; a visão **Micro** foca o script aberto e destaca as decisões funcionais do processo. É possível arrastar o fundo para mover o canvas, arrastar vértices para reorganizar a leitura, usar zoom pelo mouse e baixar uma imagem PNG da visão atual.

Depois de uma execução, o botão **Reprocessar** fica disponível. Ele usa o snapshot da execução anterior para retomar do cursor salvo, preservando as etapas ja processadas e executando somente o trecho restante.
