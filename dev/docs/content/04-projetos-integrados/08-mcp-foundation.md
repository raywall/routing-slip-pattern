# MCP e planner assistido

O MCP Gateway expoe capacidades internas do ecossistema como ferramentas consultaveis por Studio, agentes e automacoes de suporte. A ideia nao e substituir o runtime, mas criar uma camada padronizada para investigar, validar e explicar workflows sem acessar diretamente arquivos, logs ou bancos.

No `routing-slip-pattern`, o gateway MCP roda separado do endpoint REST do workflow:

| Endpoint | Uso |
| --- | --- |
| `GET /health` | Verifica se o gateway esta ativo. |
| `POST /mcp` | Recebe chamadas JSON-RPC para listar e executar tools. |

## Configuracao

```yaml
features:
  mcp_enabled: true

mcp:
  enabled: true
  bind: 127.0.0.1:9091
  mode: readonly
  auth:
    type: api_key
    env: ROUTING_SLIP_MCP_API_KEY
```

Por padrao, o modo recomendado e `readonly`. Ferramentas que alteram estado ou reprocessam execucoes exigem modo `maintenance` e implementacao controlada.

## Tools disponiveis

| Tool | Modo | O que faz |
| --- | --- | --- |
| `list_handlers` | readonly | Lista handlers e parametros principais. |
| `validate_workflow` | readonly | Valida YAML, handlers conhecidos e saltos. |
| `explain_workflow` | readonly | Resume etapas, decisoes, integracoes e pontos de controle. |
| `export_workflow` | readonly | Exporta o workflow expandido em YAML. |
| `get_execution` | readonly | Recupera snapshot por `message_id`, `correlation_id` ou `trace_id`. |
| `list_state_snapshots` | readonly | Lista snapshots por filtros simples. |
| `dry_run_step` | maintenance | Execucao controlada de etapa isolada, liberada apenas em modo de manutencao. |
| `reprocess_execution` | maintenance | Reprocessamento assistido, liberado apenas em modo de manutencao. |

## Listando tools

```bash
curl -s http://localhost:9091/mcp \
  -H 'content-type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

## Validando um workflow

```bash
curl -s http://localhost:9091/mcp \
  -H 'content-type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "validate_workflow",
      "arguments": {
        "path": "/workspace/workflows/order-processing.yaml"
      }
    }
  }'
```

## Consultando uma execucao

```bash
curl -s http://localhost:9091/mcp \
  -H 'content-type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "get_execution",
      "arguments": {
        "message_id": "order-1001"
      }
    }
  }'
```

## Segurança

- O MCP fica desligado por padrao.
- `readonly` e o modo padrao recomendado.
- `api_key` pode exigir `Authorization: Bearer <token>` ou `X-API-Key`.
- Segredos e headers sensiveis continuam sujeitos a politica de redaction do projeto.
- Tools de escrita foram registradas como contrato, mas permanecem bloqueadas ate existir implementacao operacional segura.

## Como isso ajuda

Com MCP, o Studio e agentes externos podem perguntar ao sistema:

- quais handlers existem;
- se um workflow esta estruturalmente valido;
- onde um fluxo pode parar;
- quais integracoes ele chama;
- qual foi o estado salvo de uma execucao;
- quais snapshots existem para um periodo, trace ou status.

Isso cria uma ponte entre engenharia, operacao e assistentes inteligentes sem acoplar o frontend diretamente ao runtime interno.
