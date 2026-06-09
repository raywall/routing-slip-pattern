# Mocks do case ecommerce-distributed

Cadastre estes modelos no `api-mock-service` para expor os endpoints em `https://mock.raysouz.studio`.

Os mocks foram desenhados para validar:

- enriquecimento via GraphQL connector;
- reserva de estoque;
- calculo de promessa de entrega;
- selecao de transportadora;
- emissao de documento operacional;
- separacao no centro de distribuicao;
- notificacao;
- atualizacao de status;
- publicacao de evento final.

## Cadastro

Na raiz do workspace:

```bash
make register-ecommerce-mocks
```

Ou diretamente:

```bash
BASE_URL=https://mock.raysouz.studio \
SERIAL=b7af3a9e-6d1a-4b15-9837-3e0f0b47e5b4 \
./routing-slip-pattern/cases/ecommerce-distributed/mocks/register.sh
```

O script converte os arquivos de `responses/` para o contrato administrativo do `api-mock-service`, usando `method`, `endpoint`, `responseStatus`, `responseBody` e `responseHeaders`.
