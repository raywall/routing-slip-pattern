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
| `plan_workflow` | readonly | Gera rascunho assistido a partir de descricao, evento e integracoes. |
| `suggest_handlers` | readonly | Sugere handlers conforme capacidades desejadas. |
| `generate_test_payload` | readonly | Gera payload de teste a partir do workflow. |
| `generate_bruno_collection` | readonly | Gera modelo de requisicoes para testar REST e MCP. |
| `assess_idempotency` | readonly | Aponta riscos de idempotencia e side effects. |
| `suggest_metrics` | readonly | Sugere metricas e pontos de auditoria. |
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
- Chamadas a partir do Studio usam CORS no endpoint MCP; em ambientes compartilhados, proteja o endpoint com chave ou proxy autenticado.
- Segredos e headers sensiveis continuam sujeitos a politica de redaction do projeto.
- Tools de escrita foram registradas como contrato, mas permanecem bloqueadas ate existir implementacao operacional segura.

## Uso no Studio

A aba de configuracao do Studio possui campos para `MCP endpoint` e `MCP API key`. Com isso, o usuario pode chamar tools diretamente da interface:

| Acao | Tool usada | Resultado |
| --- | --- | --- |
| Validar MCP | `validate_workflow` | Lista erros e avisos estruturais do YAML aberto. |
| Explicar MCP | `explain_workflow` | Resume etapas, decisoes, integracoes e targets. |
| Diagnosticar conectores | Leitura local do YAML | Mostra GraphQL, REST, notificacoes, retries e circuit breaker declarados. |

Essa integracao ajuda a revisar workflows maiores sem alternar entre editor, terminal e arquivos de configuracao.

## Como isso ajuda

Com MCP, o Studio e agentes externos podem perguntar ao sistema:

- quais handlers existem;
- se um workflow esta estruturalmente valido;
- onde um fluxo pode parar;
- quais integracoes ele chama;
- qual foi o estado salvo de uma execucao;
- quais snapshots existem para um periodo, trace ou status.

Isso cria uma ponte entre engenharia, operacao e assistentes inteligentes sem acoplar o frontend diretamente ao runtime interno.
