.PHONY: help submodules prepare stop run run-container studio send-rest send-kafka send-sqs test fmt compose-up compose-down compose-logs app-shell

COMPOSE ?= docker compose
APP_DIR ?= app
CONFIG ?= config.yaml
WORKFLOW ?= workflows/payment-fulfillment.yaml
PAYLOAD ?= examples/payment-event.json
GOENV := GOCACHE=$(CURDIR)/.gocache

help:
	@printf "routing-slip-pattern\n\n"
	@printf "Targets:\n"
	@printf "  make submodules    Initialize/update private internal projects\n"
	@printf "  make prepare       Start Kafka, create topic, start LocalStack and create SQS queue\n"
	@printf "  make run           Run the routing slip app locally using CONFIG and WORKFLOW\n"
	@printf "  make run-container Run the app through docker compose\n"
	@printf "  make studio        Serve the Routing Slip Studio at http://localhost:8089\n"
	@printf "  make send-rest     Publish the sample payment event through REST\n"
	@printf "  make send-kafka    Publish the sample payment event to Kafka\n"
	@printf "  make send-sqs      Publish the sample payment event to SQS/LocalStack\n"
	@printf "  make test          Run routing slip tests\n"
	@printf "  make fmt           Format routing slip Go code\n"
	@printf "  make compose-up    Start the full local stack\n"
	@printf "  make compose-down  Stop local Docker stack\n"
	@printf "  make compose-logs  Follow local Docker logs\n"

submodules:
	git submodule update --init --recursive

prepare:
	$(COMPOSE) up -d kafka kafka-init localstack localstack-init
	@printf "Kafka topic: payment-events\n"
	@printf "SQS queue:   http://localhost:4566/000000000000/payment-events\n"
	@printf "\nReady. Run locally with: make run CONFIG=config.yaml WORKFLOW=workflows/payment-fulfillment.yaml\n"

stop:
	$(COMPOSE) down

run:
	$(GOENV) go -C $(APP_DIR) run . --config ../$(CONFIG) --workflow ../$(WORKFLOW)

run-container:
	$(COMPOSE) up --build routing-slip-app

studio:
	python3 -m http.server 8089 --directory studio

send-rest:
	curl -fsS -X POST http://localhost:8088/process \
		-H 'Content-Type: application/json' \
		--data-binary @$(PAYLOAD)

send-kafka:
	$(GOENV) go -C $(APP_DIR) run ./cmd/publish-event --config ../$(CONFIG) --payload ../$(PAYLOAD) --target kafka

send-sqs:
	$(GOENV) go -C $(APP_DIR) run ./cmd/publish-event --config ../$(CONFIG) --payload ../$(PAYLOAD) --target sqs

test:
	$(GOENV) go -C $(APP_DIR) test ./...

fmt:
	$(GOENV) go -C $(APP_DIR) fmt ./...

compose-up:
	$(COMPOSE) up --build

compose-down:
	$(COMPOSE) down

compose-logs:
	$(COMPOSE) logs -f

app-shell:
	$(COMPOSE) run --rm routing-slip-app sh
