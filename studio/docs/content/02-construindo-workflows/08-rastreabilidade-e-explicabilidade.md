# Rastreabilidade e explicabilidade

Rastreabilidade mostra onde uma execucao passou. Explicabilidade mostra por que ela tomou determinado caminho.

O Routing Slip registra informacoes que ajudam os times a responder perguntas como:

- qual payload entrou?
- qual etapa falhou?
- qual regra parou o fluxo?
- qual integracao demorou mais?
- qual etapa foi reprocessada?
- o que ja tinha sido concluido?

## Identificadores importantes

| Campo | Papel |
| --- | --- |
| `message_id` | Identifica a execucao e o snapshot. |
| `correlation_id` | Agrupa eventos do mesmo processo de negocio. |
| `trace_id` | Agrupa chamadas tecnicas ponta a ponta. |
| `span_id` | Identifica uma etapa ou chamada dentro do trace. |
| `cursor` | Indica a proxima etapa a executar. |

## Historico por etapa

Cada etapa pode registrar:

- nome do handler;
- horario de inicio;
- duracao;
- status;
- tentativa;
- trace/span;
- se foi pulada ou redirecionada.

```json
{
  "Step": "graphql_enrich",
  "Status": "success",
  "Duration": 180000000,
  "TraceID": "4bf92f3577b34da6a3ce929d0e0e4736",
  "Attempt": 1
}
```

## Explicabilidade no desenho

Use `id` nos steps, `audit` em pontos importantes, `assert` para regras obrigatorias e `condition` para paradas funcionais esperadas. Assim, a execucao fica compreensivel tanto durante o teste quanto em producao.

