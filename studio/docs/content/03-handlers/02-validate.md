# Validate

Use `validate` para garantir que campos essenciais existem antes de executar efeitos externos.

```yaml
- name: validate
  params:
    required:
      - pedido_id
      - correlation_id
```

Por padrao, campos ausentes geram erro e o workflow respeita a `error_policy`.

```yaml
- name: validate
  params:
    required:
      - pedido_id
      - comprador.id
      - itens.0.sku
      - entrega.endereco.cep
    stop_on_failure: true
```

Para apenas registrar a falha e continuar:

```yaml
- name: validate
  params:
    required:
      - metadados.origem
    stop_on_failure: false
```

O handler grava `validation_passed: true` quando tudo passa e `validation_error` quando faltam campos.
