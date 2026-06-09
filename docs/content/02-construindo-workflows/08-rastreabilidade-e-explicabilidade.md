---
sidebar_position: 8
sidebar_label: "Rastreabilidade e explicabilidade"
---

# Rastreabilidade e explicabilidade

Rastreabilidade mostra onde uma execução passou. Explicabilidade mostra por que ela tomou determinado caminho.

O Routing Slip registra informações que ajudam os times a responder perguntas como:

- qual payload entrou?
- qual etapa falhou?
- qual regra parou o fluxo?
- qual integração demorou mais?
- qual etapa foi reprocessada?
- o que ja tinha sido concluído?

## Identificadores importantes

| Campo | Papel |
| --- | --- |
| `message_id` | Identifica a execução e o snapshot. |
| `correlation_id` | Agrupa eventos do mesmo processo de negocio. Se ausente, o runtime gera um UUID v4. |
| `trace_id` | Agrupa chamadas técnicas ponta a ponta. |
| `span_id` | Identifica uma etapa ou chamada dentro do trace. |
| `cursor` | Indica a proxima etapa a executar. |

## Histórico por etapa

Cada etapa pode registrar:

- nome do handler;
- horário de inicio;
- duração;
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

Use `id` nos steps, `audit` em pontos importantes, `assert` para regras obrigatórias e `condition` para paradas funcionais esperadas. Assim, a execução fica compreensível tanto durante o teste quanto em produção.
