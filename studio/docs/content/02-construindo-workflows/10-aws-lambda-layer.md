# AWS Lambda Layer

O `routing-slip-pattern` pode ser usado em Lambdas Go. A proposta inicial e publicar uma Lambda Layer com configuracoes, workflows e assets compartilhados em `/opt/routing-slip`, deixando o codigo da Lambda focado em receber o evento, carregar o workflow e acionar o router.

Em Go, a Lambda e compilada em um binario. Por isso, a layer nao substitui o `go.mod` da funcao. O codigo ainda importa os pacotes do framework no build, enquanto a layer entrega arquivos usados em runtime.

## Terraform

O projeto fornece um Terraform em:

```text
infra/lambda-layer
```

Publicacao basica:

```bash
cd infra/lambda-layer
terraform init
terraform apply \
  -var='aws_region=us-east-1' \
  -var='layer_name=routing-slip-pattern-framework'
```

Depois de criada, use o ARN versionado da layer na Lambda:

```hcl
layers = [
  "arn:aws:lambda:REGION:ACCOUNT:layer:routing-slip-pattern-framework:1"
]
```

Se existirem layers internas para certificados, extensoes ou bibliotecas corporativas, preencha `extra_layer_arns`:

```hcl
extra_layer_arns = [
  "arn:aws:lambda:REGION:ACCOUNT:layer:certificados-internos:1"
]
```

## Estrutura em runtime

Por padrao, a layer publica:

```text
/opt/routing-slip/
  config/config.yaml
  workflows/workflow.yaml
```

A Lambda pode usar variaveis de ambiente para localizar estes arquivos:

| Variavel | Exemplo |
| --- | --- |
| `ROUTING_SLIP_CONFIG_PATH` | `/opt/routing-slip/config/config.yaml` |
| `ROUTING_SLIP_WORKFLOW_PATH` | `/opt/routing-slip/workflows/workflow.yaml` |

## Simulação local via Docker

O `docker-compose.yml` do laboratório usa uma imagem local que compila o runtime Go como `/var/task/bootstrap` e monta a layer em `/opt/routing-slip`. A porta REST continua exposta em `8088`, permitindo testar a Lambda simulada com a mesma chamada usada no runtime local:

```bash
make prepare
curl -X POST http://localhost:8088/process \
  -H "Content-Type: application/json" \
  -d @routing-slip-pattern/examples/payment-event.json
```

Na simulação, o serviço mantém o nome `routing-slip-app` para compatibilidade com o Makefile, mas o container executa com hostname `routing-slip-lambda`.

## Exemplo Go

```go
package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/raywall/routing-slip-pattern/handlers"
	"github.com/raywall/routing-slip-pattern/slip"
	"gopkg.in/yaml.v3"
)

type workflowFile struct {
	Name          string         `yaml:"name"`
	MessageIDPath string         `yaml:"message_id_path"`
	Steps         []slip.StepDef `yaml:"steps"`
}

func main() {
	data, err := os.ReadFile(env("ROUTING_SLIP_WORKFLOW_PATH", "/opt/routing-slip/workflows/workflow.yaml"))
	if err != nil {
		panic(err)
	}

	var workflow workflowFile
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		panic(err)
	}

	router := slip.NewRouter(slip.WithErrorPolicy(slip.StopOnError))
	router.MustRegister(handlers.ValidationHandler{})
	router.MustRegister(handlers.EnrichmentHandler{})
	router.MustRegister(handlers.AuditHandler{})

	lambda.Start(func(ctx context.Context, payload map[string]any) (map[string]any, error) {
		messageID, _ := payload[workflow.MessageIDPath].(string)
		if messageID == "" {
			messageID = "lambda-request"
		}

		msg := slip.NewMessage(messageID, payload)
		msg.Headers["workflow"] = workflow.Name
		msg.AttachSlip(workflow.Steps)

		err := router.Process(ctx, msg)
		return map[string]any{
			"message_id":      msg.ID,
			"remaining_steps": msg.RemainingSteps(),
			"payload":         msg.Payload,
			"errors":          msg.Errors,
		}, err
	})
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
```

## Quando usar

Use a layer quando varios times precisam compartilhar a mesma base operacional de workflows, configuracoes e assets. Para execucao com retomada do ponto de falha, combine a Lambda com state store persistente, como DynamoDB.
