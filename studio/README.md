# Routing Slip Studio

Interface local para editar, validar e testar workflows YAML do
`routing-slip-pattern`.

## Executar

Na raiz do projeto:

```bash
make studio
```

Acesse:

```text
http://localhost:8089
```

Tambem funciona com qualquer servidor estatico apontando para esta pasta.

## Recursos

- Editor YAML para workflows.
- Sidebar redimensionavel.
- Paineis recolhiveis para configuracao/runtime e payload.
- Indentacao com Tab e Shift+Tab no editor YAML.
- Numeros de linha sincronizados com o scroll do editor.
- Comentario/descomentario de bloco com `Cmd+/` ou `Ctrl+/`.
- Clique em logs de etapa para focar o trecho equivalente no YAML.
- Logs agrupados por fase/step para facilitar a leitura da execucao.
- Estado salvo no IndexedDB local, com fallback para localStorage.
- Lint estrutural dos handlers suportados.
- Editor JSON para payload de entrada.
- Simulacao local da execucao passo a passo.
- Logs por etapa com payload, cursor, historico e erros.
- Envio opcional para o endpoint REST real do `routing-slip-pattern`.
- Toggle para chamadas reais de `graphql_enrich` e `rest_call`.

## Observacoes

O Studio valida a estrutura conhecida do workflow, mas nao substitui a execucao
real do app Go. Use a simulacao para acelerar a criacao do script e depois teste
o fluxo real com o app iniciado via:

```bash
go run ./app --config ./config.yaml --workflow ./workflows/payment-fulfillment.yaml
```
