# Visão geral

Handlers sao unidades pequenas e combináveis. Cada handler recebe o payload atual, seus `params` e decide se altera o payload, interrompe o fluxo, registra informações ou chama uma integração.

| Handler | Papel |
|---|---|
| `validate` | Verifica campos obrigatórios. |
| `condition` | Para o fluxo de forma funcional quando uma regra nao bate. |
| `assert` | Falha o workflow quando uma regra obrigatória nao e atendida. |
| `compute` | Calcula e grava valores no payload. |
| `cel` | Avalia expressões CEL e decide erro, salto, continuidade ou parada. |
| `filter_array` | Remove itens de arrays que nao atendem a uma condição. |
| `array_transform` | Filtra arrays, altera campos dos itens e transforma arrays aninhados. |
| `jump_if` | Altera o cursor para uma etapa posterior. |
| `enrich` | Injeta dados estáticos no payload. |
| `transform` | Normaliza texto. |
| `graphql_enrich` | Enriquece via GraphQL Connector. |
| `rest_call` | Chama uma API REST e salva a resposta. |
| `audit` | Registra evidencia funcional. |
| `notify` | Simula envio de notificação. |

O runtime suporta CEL por meio do handler `cel`, usando `cel-go`. Use validações declarativas para regras simples e CEL quando a expressão deixar a regra mais clara.
