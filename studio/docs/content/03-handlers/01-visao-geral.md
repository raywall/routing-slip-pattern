# Visao geral dos handlers

Handlers sao unidades pequenas e combinaveis. Cada handler recebe o payload atual, seus `params` e decide se altera o payload, interrompe o fluxo, registra informacoes ou chama uma integracao.

| Handler | Papel |
|---|---|
| `validate` | Verifica campos obrigatorios. |
| `condition` | Para o fluxo de forma funcional quando uma regra nao bate. |
| `assert` | Falha o workflow quando uma regra obrigatoria nao e atendida. |
| `compute` | Calcula e grava valores no payload. |
| `cel` | Avalia expressoes CEL e decide erro, salto, continuidade ou parada. |
| `filter_array` | Remove itens de arrays que nao atendem a uma condicao. |
| `jump_if` | Altera o cursor para uma etapa posterior. |
| `enrich` | Injeta dados estaticos no payload. |
| `transform` | Normaliza texto. |
| `graphql_enrich` | Enriquece via GraphQL Connector. |
| `rest_call` | Chama uma API REST e salva a resposta. |
| `audit` | Registra evidencia funcional. |
| `notify` | Simula envio de notificacao. |

O runtime suporta CEL por meio do handler `cel`, usando `cel-go`. Use validacoes declarativas para regras simples e CEL quando a expressao deixar a regra mais clara.
