Use `enrich` para adicionar dados ao payload.

```yaml
- name: enrich
  params:
    data:
      origem: CHECKOUT_ONLINE
      prioridade: NORMAL
```

Com prefixo:

```yaml
- name: enrich
  params:
    prefix: meta_
    data:
      origem: CHECKOUT_ONLINE
```

Use `transform` para normalizar texto.

```yaml
- name: transform
  params:
    field: comprador.email
    operation: lowercase
    target: comprador_email_normalizado
```

Operacoes suportadas: `uppercase`, `lowercase`, `trim`, `prefix:<valor>` e `suffix:<valor>`.
