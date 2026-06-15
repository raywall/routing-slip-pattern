# Exemplos de uso do framework

O Studio é usado para arquitetar e testar workflows. A execução em produção
acontece pelo runtime importado pela aplicação.

| Exemplo | Objetivo |
|---|---|
| `importable-rest` | Servidor para ECS, EKS, VM ou local, com metrics agent e MCP. |
| `importable-lambda` | Handler AWS Lambda que processa payloads com o runtime. |
| `aws-sources` | Configuração e workflow recuperados de origens AWS. |

```bash
cd examples/importable-rest
go run .
curl -X POST http://localhost:8088/process \
  -H 'content-type: application/json' \
  -d '{"event_id":"evt-001","product_id":"PRD-100"}'
```
