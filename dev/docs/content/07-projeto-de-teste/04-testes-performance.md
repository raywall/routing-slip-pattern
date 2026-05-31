---
sidebar_position: 4
sidebar_label: "Testes de performance"
---

# Testes de performance

Os testes de performance medem como o ecossistema se comporta quando o volume de eventos aumenta. O foco não é apenas medir tempo total, mas entender gargalos por etapa, integração, handler e dependência externa.

## Métricas

| Métrica | Objetivo |
|---|---|
| Throughput | Quantidade de processamentos por segundo. |
| Tempo total | Duração completa de cada workflow. |
| Tempo por etapa | Identificar gargalos de validação, enriquecimento ou integração. |
| p50, p90, p95, p99 | Avaliar distribuição de latência. |
| Retries | Medir impacto de instabilidade externa. |
| Circuit breakers | Entender quando uma dependência ficou indisponível. |
| Uso de memória/CPU | Avaliar capacidade local ou do container. |
| Fila de métricas | Detectar atraso de observabilidade. |

## Cenários

| Cenário | Comando base | Resultado esperado |
|---|---|---|
| Carga REST curta | `make ecommerce-load COUNT=25` | Validar estabilidade inicial. |
| Carga REST concorrente | `make ecommerce-performance COUNT=100 CONCURRENCY=8` | Observar latência, throughput, p95 e p99. |
| Geração NDJSON | `make ecommerce-events-file COUNT=1000` | Criar massa para Kafka/SQS. |
| GraphQL sob repetição | Bruno ou script externo | Medir tempo do conector e APIs mockadas. |
| Métricas sob carga | Dashboard e API de métricas | Confirmar ingestão e consulta por trace/correlation. |

## Procedimento sugerido

1. Subir a stack com `make prepare`.
2. Garantir mocks cadastrados.
3. Executar a carga com `make ecommerce-performance COUNT=100 CONCURRENCY=8`.
5. Consultar métricas por workflow, step, status e `trace_id`.
6. Registrar resultados em `cases/ecommerce-distributed/results/`.

## Resultados esperados

| Resultado | Critério |
|---|---|
| Sem perda de rastreabilidade | Todo processamento possui `correlation_id` e `trace_id`. |
| Latência explicável | Etapas lentas aparecem em métricas e logs. |
| Falhas classificáveis | Erros indicam etapa, handler e integração. |
| Observabilidade útil | Dashboard permite localizar processos específicos. |
| Reprocessamento comparável | Execução original e retomada podem ser comparadas. |

## Resultado registrado

O runner grava automaticamente:

```text
cases/ecommerce-distributed/results/latest-performance.json
```

O arquivo contém total, concluídos, falhos, média, p95, p99, tempo máximo e throughput por segundo.
