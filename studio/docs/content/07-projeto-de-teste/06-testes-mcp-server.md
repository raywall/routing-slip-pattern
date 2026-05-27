# Testes do MCP Server

Os testes do MCP Server validam se o runtime consegue expor informações úteis para consulta, validação e investigação sem executar o workflow nem alterar estado operacional.

O objetivo é confirmar que o MCP funciona como uma camada segura de leitura, explicação e apoio à análise.

## Dados disponíveis para consulta

| Dado | Origem | Uso |
|---|---|---|
| Handlers registrados | Runtime | Confirmar quais capacidades o workflow pode usar. |
| Workflow carregado | Arquivo YAML informado no runtime | Validar estrutura, handlers, saltos e composição. |
| Snapshots de execução | State store | Consultar processos por `message_id`, `correlation_id`, `trace_id`, workflow ou status. |
| Histórico de etapas | Snapshot | Entender cursor, etapas concluídas, falhas e retomadas. |
| Métricas sugeridas | Planner MCP | Avaliar pontos de observabilidade recomendados. |
| Riscos de idempotência | Planner MCP | Identificar side effects e pontos sensíveis do workflow. |

## Requisições Bruno

| Requisição | Tool | Resultado esperado |
|---|---|---|
| `Listar tools MCP` | `tools/list` | Lista de tools disponíveis e metadados de leitura. |
| `Validar workflow ecommerce` | `validate_workflow` | `valid: true` para o workflow carregado. |
| `Explicar workflow ecommerce` | `explain_workflow` | Lista das etapas, handlers, integrações e controles. |
| `Sugerir metricas ecommerce` | `suggest_metrics` | Métricas e pontos de auditoria recomendados. |

## Consultas manuais

Listar tools:

```bash
curl -s http://localhost:9091/mcp \
  -H 'content-type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

Validar workflow carregado:

```bash
curl -s http://localhost:9091/mcp \
  -H 'content-type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "validate_workflow",
      "arguments": {}
    }
  }'
```

Consultar execução por `correlation_id`:

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
        "correlation_id": "839e2b76-0aa1-47dc-96b1-67f41b73c795"
      }
    }
  }'
```

Consultar snapshots por status:

```bash
curl -s http://localhost:9091/mcp \
  -H 'content-type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "id": 4,
    "method": "tools/call",
    "params": {
      "name": "list_state_snapshots",
      "arguments": {
        "workflow": "ecommerce-order-fulfillment",
        "status": "failed",
        "limit": 10
      }
    }
  }'
```

## Validações esperadas

| Validação | Critério |
|---|---|
| Segurança | Tools de escrita continuam bloqueadas em modo `readonly`. |
| Estrutura | `validate_workflow` identifica handlers desconhecidos, YAML inválido e saltos inconsistentes. |
| Explicabilidade | `explain_workflow` descreve etapas, integrações e controles. |
| Consulta operacional | `get_execution` recupera snapshot por identificador quando existir. |
| Busca granular | `list_state_snapshots` filtra por workflow, status, correlation e trace. |
| Observabilidade | `suggest_metrics` recomenda métricas coerentes com os steps do workflow. |

## Resultados

| Teste | Resultado esperado | Resultado observado | Evidência |
|---|---|---|---|
| Listar tools | Tools retornadas com sucesso. | A preencher. | A preencher. |
| Validar workflow | `valid: true`. | A preencher. | A preencher. |
| Explicar workflow | Etapas e integrações descritas. | A preencher. | A preencher. |
| Consultar execução | Snapshot encontrado após execução. | A preencher. | A preencher. |
| Listar snapshots | Lista filtrada conforme argumentos. | A preencher. | A preencher. |
| Sugerir métricas | Métricas recomendadas. | A preencher. | A preencher. |

Use a coluna de evidência para inserir imagens ou links para capturas depois dos testes.
