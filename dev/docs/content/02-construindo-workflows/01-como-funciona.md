# Como o workflow funciona

Um workflow e um contrato declarativo. Ele descreve a sequencia de handlers que processa um payload, quais campos identificam a mensagem, qual politica de erro deve ser usada e quais etapas podem enriquecer, validar, transformar, auditar ou chamar integracoes.

O runtime executa a lista em ordem. A cada etapa, ele atualiza o cursor, registra historico, emite metricas e salva estado quando o state store esta habilitado.

```mermaid
flowchart TD
  A[Receber evento] --> B[Resolver message_id]
  B --> C[Carregar snapshot se existir]
  C --> D[Executar etapa do cursor]
  D --> E{Etapa concluiu?}
  E -- Sim --> F[Salvar historico e cursor]
  E -- Nao --> G[Registrar erro]
  G --> H{Politica de erro}
  H -- stop --> I[Salvar cursor da falha]
  H -- continue --> F
  H -- skip --> F
  F --> J{Ha mais etapas?}
  J -- Sim --> D
  J -- Nao --> K[Workflow concluido]
```

## Quando usar

Use o Routing Slip quando o processo:

- possui multiplas etapas;
- precisa consultar dados externos;
- precisa ser rastreavel e explicavel;
- deve continuar do ponto de falha;
- pode crescer e precisa ser dividido em partes;
- deve gerar metricas por etapa.

## Quando evitar

Se o processamento e uma unica operacao simples, sem reprocessamento, sem regras e sem integracoes, um endpoint comum pode ser suficiente. O Routing Slip mostra mais valor quando o fluxo tem variacao, risco operacional ou necessidade de visibilidade.

