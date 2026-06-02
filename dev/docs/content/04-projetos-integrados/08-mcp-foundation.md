---
sidebar_position: 7
sidebar_label: "MCP e planner assistido"
---

# MCP e planner assistido

O MCP Gateway expõe capacidades internas do ecossistema como ferramentas consultáveis por Studio, agentes e automações de suporte. A ideia nao e substituir o runtime, mas criar uma camada padronizada para investigar, validar e explicar workflows sem acessar diretamente arquivos, logs ou bancos.

No `routing-slip-pattern`, o gateway MCP roda separado do endpoint REST do workflow:

| Endpoint | Uso |
| --- | --- |
| `GET /health` | Verifica se o gateway esta ativo. |
| `POST /mcp` | Recebe chamadas JSON-RPC para listar e executar tools. |

## Configuração

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

Por padrão, o modo recomendado e `readonly`. Ferramentas que alteram estado ou reprocessam execuções exigem modo `maintenance` e implementação controlada.

## Tools disponíveis

| Tool | Modo | O que faz |
| --- | --- | --- |
| `list_handlers` | readonly | Lista handlers e parâmetros principais. |
| `validate_workflow` | readonly | Valida YAML, handlers conhecidos e saltos. |
| `explain_workflow` | readonly | Resume etapas, decisões, integrações e pontos de controle. |
| `export_workflow` | readonly | Exporta o workflow expandido em YAML. |
| `get_execution` | readonly | Recupera snapshot por `message_id`, `correlation_id` ou `trace_id`. |
| `list_state_snapshots` | readonly | Lista snapshots por filtros simples. |
| `plan_workflow` | readonly | Gera rascunho assistido a partir de descrição, evento e integrações. |
| `generate_workflow_from_business_rules` | readonly | Gera YAML e payload base a partir de regras de negócio. |
| `validate_workflow_against_business_rules` | readonly | Verifica aderência do workflow às regras ativas informadas. |
| `suggest_handlers` | readonly | Sugere handlers conforme capacidades desejadas. |
| `generate_test_payload` | readonly | Gera payload de teste a partir do workflow. |
| `generate_bruno_collection` | readonly | Gera modelo de requisições para testar REST e MCP. |
| `assess_idempotency` | readonly | Aponta riscos de idempotência e side effects. |
| `suggest_metrics` | readonly | Sugere métricas e pontos de auditoria. |
| `dry_run_step` | maintenance | Execução controlada de etapa isolada, liberada apenas em modo de manutenção. |
| `reprocess_execution` | maintenance | Reprocessamento assistido, liberado apenas em modo de manutenção. |

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

## Consultando uma execução

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

- O MCP fica desligado por padrão.
- `readonly` e o modo padrão recomendado.
- `api_key` pode exigir `Authorization: Bearer <token>` ou `X-API-Key`.
- Chamadas a partir do Studio usam CORS no endpoint MCP; em ambientes compartilhados, proteja o endpoint com chave ou proxy autenticado.
- Segredos e headers sensíveis continuam sujeitos a politica de redaction do projeto.
- Tools de escrita foram registradas como contrato, mas permanecem bloqueadas ate existir implementação operacional segura.

## Uso no Studio

A aba de configuração do Studio possui campos para `MCP endpoint` e `MCP API key`. Com isso, o usuário pode chamar tools diretamente da interface:

| Ação | Tool usada | Resultado |
| --- | --- | --- |
| Validar MCP | `validate_workflow` | Lista erros e avisos estruturais do YAML aberto. |
| Explicar MCP | `explain_workflow` | Resume etapas, decisões, integrações e targets. |
| Diagnosticar conectores | Leitura local do YAML | Mostra GraphQL, REST, notificações, retries e circuit breaker declarados. |

Essa integração ajuda a revisar workflows maiores sem alternar entre editor, terminal e arquivos de configuração.

## Como isso ajuda

Com MCP, o Studio e agentes externos podem perguntar ao sistema:

- quais handlers existem;
- se um workflow esta estruturalmente valido;
- onde um fluxo pode parar;
- quais integrações ele chama;
- qual foi o estado salvo de uma execução;
- quais snapshots existem para um período, trace ou status.

Isso cria uma ponte entre engenharia, operação e assistentes inteligentes sem acoplar o frontend diretamente ao runtime interno.
