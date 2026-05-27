# Estratégia com Routing Slip

A estratégia adotada foi dividir o processo em workflows menores, cada um com uma responsabilidade clara. O workflow principal funciona como a trilha de orquestração, enquanto os subfluxos tratam contexto, entrega e operações.

## Organização

| Arquivo | Responsabilidade |
|---|---|
| `order-fulfillment-main.yaml` | Receber evento, validar entrada, conectar subfluxos e auditar conclusão. |
| `order-context.yaml` | Buscar contexto via GraphQL, validar dados e filtrar estoque. |
| `reserve-and-delivery.yaml` | Reservar estoque, calcular promessa e selecionar transportadora. |
| `operations-and-notification.yaml` | Acionar operações, notificar cliente, atualizar status e publicar evento final. |

## Funcionalidades utilizadas

| Funcionalidade | Uso no projeto de teste |
|---|---|
| `workflow_ref` | Divide o fluxo longo em arquivos menores e mantém execução contínua. |
| `validate` | Garante campos mínimos antes de qualquer integração. |
| `graphql_enrich` | Consulta o `go-graphql-connector` para montar o contexto. |
| `assert` | Trava o fluxo quando critérios obrigatórios não são atendidos. |
| `filter_array` | Remove itens sem disponibilidade de estoque. |
| `compute` | Cria flags derivadas, como pedido de alta prioridade. |
| `rest_call` | Aciona serviços externos de reserva, entrega, operação e notificação. |
| `audit` | Registra marcos funcionais e melhora explicabilidade. |
| `resilience` | Define retry, backoff, jitter e política de falha por etapa. |
| State store | Preserva cursor, payload, histórico e estado granular. |
| MCP | Valida, explica e sugere métricas sem executar efeitos colaterais. |
| Studio | Permite editar, simular, diagnosticar conectores e visualizar logs por etapa. |

## Rastreabilidade

Cada processamento deve possuir um `correlation_id` único em formato UUID. O gerador de eventos cria um UUID v4 novo para cada execução, evitando reutilizar o identificador do payload base.

O `trace_id` é criado e propagado pelo runtime. Ele acompanha chamadas para o GraphQL connector, APIs externas e eventos de métricas, permitindo investigar a execução ponta a ponta.

## Resiliência

As integrações obrigatórias usam retry e parada em falha. A notificação é opcional e pode continuar o fluxo quando falha, desde que essa decisão esteja declarada no YAML.

```yaml
resilience:
  retry:
    attempts: 3
    backoff: exponential
    initial_interval_ms: 200
    max_interval_ms: 1500
    jitter: true
  on_failure:
    action: stop
```

## MCP no projeto de teste

A coleção Bruno possui chamadas para:

- listar tools MCP;
- validar o workflow carregado;
- explicar o workflow expandido;
- sugerir métricas e pontos de auditoria.

Essas chamadas ajudam a confirmar se o workflow está compreensível para ferramentas externas e se o desenho operacional está suficientemente explicável.
