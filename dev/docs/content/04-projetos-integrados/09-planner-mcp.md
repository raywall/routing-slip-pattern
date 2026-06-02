---
sidebar_position: 8
sidebar_label: "Criação assistida"
---

# Planner assistido por MCP

O planner assistido por MCP ajuda a transformar descrição, evento, endpoints e pistas de sistemas existentes em um rascunho de workflow. Ele nao executa steps, nao cria arquivos e nao faz merge automático. O objetivo e acelerar a criação com explicabilidade.

## Tools do planner

| Tool | O que entrega |
| --- | --- |
| `plan_workflow` | Rascunho YAML, payload de teste, requisições Bruno, métricas, auditoria e riscos. |
| `generate_workflow_from_business_rules` | Gera workflow e payload base a partir das regras de negocio ativas. |
| `validate_workflow_against_business_rules` | Verifica se o workflow cobre as regras ativas informadas. |
| `suggest_handlers` | Sugere handlers com base em capacidades descritas. |
| `generate_test_payload` | Gera payload coerente com o workflow informado. |
| `generate_bruno_collection` | Gera modelo textual de requisições para testar REST e MCP. |
| `assess_idempotency` | Aponta steps com risco de side effect ou dados externos mutáveis. |
| `suggest_metrics` | Sugere métricas e pontos de auditoria. |

## Exemplo de planejamento

```bash
curl -s http://localhost:9091/mcp \
  -H 'content-type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "id": 10,
    "method": "tools/call",
    "params": {
      "name": "plan_workflow",
      "arguments": {
        "name": "Catalog Sync",
        "description": "Recebe evento de catalogo, consulta API REST de produto e audita o resultado",
        "required_fields": ["correlation_id", "product_id"],
        "endpoints": [
          {
            "name": "product-api",
            "method": "GET",
            "url": "https://api.example.test/products/{product_id}"
          }
        ]
      }
    }
  }'
```

A resposta traz:

- `yaml`: rascunho do workflow;
- `test_payload`: payload inicial para teste;
- `bruno_requests`: requisições sugeridas;
- `idempotency`: riscos e recomendações;
- `metrics`: métricas relevantes;
- `audit_points`: eventos de auditoria sugeridos;
- `decision_notes`: explicação das decisões tomadas;
- `requires_review: true`.

## Restrições de segurança

O planner sempre trabalha em modo assistivo:

- nao grava arquivos;
- nao executa integrações;
- nao reprocessa execuções;
- nao substitui revisão humana;
- sempre marca o resultado como `requires_review`.

## Como usar no desenho de workflows

1. Descreva o fluxo em linguagem natural.
2. Informe o evento de entrada e campos obrigatórios.
3. Liste endpoints ou integrações conhecidas.
4. Rode `plan_workflow`.
5. Valide o YAML com `validate_workflow`.
6. Revise riscos de idempotência.
7. Ajuste nomes, paths e regras funcionais.
8. Teste pelo Studio ou endpoint REST.

O planner e propositalmente conservador. Ele prefere criar um fluxo simples, auditável e seguro, deixando regras complexas para revisão e evolução incremental.

## Gerar a partir de regras de negocio

Quando o arquivo aberto no Studio possui regras de negocio, o botão **Gerar por regras** chama `generate_workflow_from_business_rules`. A resposta traz o YAML sugerido, um payload base e uma tabela de cobertura para mostrar quais regras originaram quais steps.

```bash
curl -s http://localhost:9091/mcp \
  -H 'content-type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "id": 11,
    "method": "tools/call",
    "params": {
      "name": "generate_workflow_from_business_rules",
      "arguments": {
        "workflow_name": "order-review",
        "business_rules": [
          {
            "rule_id": "order_total_positive",
            "status": "ACTIVE",
            "execution_order": 1,
            "human_context": {
              "name": "Total positivo",
              "description": "O campo {order.total} deve existir antes da aprovacao."
            },
            "technical_metadata": {
              "observability": {
                "custom_metric": ["routing_slip.order.total_checked"],
                "log_markers": ["total-check"]
              }
            }
          }
        ]
      }
    }
  }'
```

O rascunho usa `validate`, `log`, `cel`, `audit` e `datadog_metric` quando a regra possui métricas declaradas. As expressões CEL geradas são pontos iniciais: revise paths, operadores e efeitos externos antes de salvar como versão final.

## Lint por regras ativas

O Studio também valida se o workflow cobre as regras `ACTIVE` do projeto. O lint alerta quando:

- uma regra ativa não aparece em nenhum step;
- campos citados em `{path}` não aparecem no workflow;
- uma métrica declarada não possui `datadog_metric`;
- um marcador de log declarado não possui `log`;
- uma dependência de regra aponta para uma regra inexistente ou inativa.

Essa validação ajuda a manter rastreabilidade entre intenção de negócio, script YAML e observabilidade.
