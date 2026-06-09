# Lambda Layer do routing-slip-pattern

Este Terraform publica uma Lambda Layer com assets do `routing-slip-pattern` e, opcionalmente, cria uma Lambda Go exemplo anexando a layer.

O conteudo da pasta `layer` fica disponivel em `/opt` durante a execucao da Lambda. Por padrao, a estrutura publicada e:

```text
/opt/routing-slip/
  config/config.yaml
  workflows/workflow.yaml
```

## Publicar apenas a layer

```bash
terraform init
terraform apply \
  -var='aws_region=us-east-1' \
  -var='layer_name=routing-slip-pattern-framework'
```

Depois da criacao, copie os outputs para usar em outras Lambdas:

```hcl
routing_slip_layer_arn = "arn:aws:lambda:REGION:ACCOUNT:layer:routing-slip-pattern-framework:1"
```

## Criar uma Lambda exemplo

Compile sua Lambda Go como `bootstrap` e gere um `.zip`:

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap ./cmd/lambda
zip function.zip bootstrap
```

Depois aplique:

```bash
terraform apply \
  -var='create_example_lambda=true' \
  -var='example_lambda_package_file=./function.zip' \
  -var='extra_layer_arns=["arn:aws:lambda:REGION:ACCOUNT:layer:minha-layer-interna:1"]'
```

`extra_layer_arns` e o ponto para preencher ARNs de layers internas, certificados, extensoes ou dependencias corporativas.

## Observacao sobre Go e Lambda Layers

Uma Lambda Go e compilada em um binario. Por isso, a layer nao injeta pacotes Go no build automaticamente. O uso recomendado e:

1. importar `github.com/raywall/routing-slip-pattern/app/slip` e `github.com/raywall/routing-slip-pattern/app/handlers` no codigo da Lambda;
2. anexar esta layer para disponibilizar configuracoes, workflows e assets em `/opt/routing-slip`;
3. usar `ROUTING_SLIP_CONFIG_PATH` e `ROUTING_SLIP_WORKFLOW_PATH` para localizar os arquivos em runtime.
