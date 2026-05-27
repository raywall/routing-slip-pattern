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

Os arquivos usam campos `request` e `response` para deixar claro metodo, path, status e payload esperado.
