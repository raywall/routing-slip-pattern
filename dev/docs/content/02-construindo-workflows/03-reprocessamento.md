O routing slip guarda o cursor da execucao. Quando uma etapa falha, o estado pode ser persistido e uma execucao futura retoma a partir da etapa que falhou.

Esse comportamento evita repetir etapas anteriores que ja produziram efeito, como emissao fiscal, envio de notificacao ou atualizacao de inventario.

Para etapas com efeito externo, combine reprocessamento com idempotencia por:

- message_id;
- step;
- attempt;
- correlation_id.
