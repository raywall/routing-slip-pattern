.PHONY: help submodules prepare stop run test fmt compose-up compose-down compose-logs metrics-up metrics-down app-shell

COMPOSE ?= docker compose
APP_DIR ?= app
GOENV := GOCACHE=$(CURDIR)/.gocache

help:
	@printf "routing-slip-pattern\n\n"
	@printf "Targets:\n"
	@printf "  make submodules    Initialize/update private internal projects\n"
	@printf "  make prepare       Start metrics, DynamoDB, webview and GraphQL mock services\n"
	@printf "  make run           Run the routing slip demos locally\n"
	@printf "  make test          Run routing slip tests\n"
	@printf "  make fmt           Format routing slip Go code\n"
	@printf "  make compose-up    Start local metrics dependencies and run app demo container\n"
	@printf "  make compose-down  Stop local Docker stack\n"
	@printf "  make compose-logs  Follow local Docker logs\n"
	@printf "  make metrics-up    Start only metrics stack services\n"
	@printf "  make metrics-down  Stop local Docker stack\n"

submodules:
	git submodule update --init --recursive

prepare: submodules
	$(COMPOSE) up -d --build dynamodb dynamodb-init metrics-service metrics-agent metrics-webview mock-external-api go-graphql-connector
	@printf "Waiting for metrics service"; \
	until curl -fsS http://localhost:8080/health >/dev/null 2>&1; do printf "."; sleep 1; done; printf " ok\n"
	@printf "Waiting for GraphQL connector"; \
	until curl -fsS http://localhost:8090/health >/dev/null 2>&1; do printf "."; sleep 1; done; printf " ok\n"
	@printf "\nReady. Run scenarios with: make run\n"
	@printf "Metrics dashboard: http://localhost:5173\n"

stop:
	$(COMPOSE) down

run:
	$(GOENV) go -C $(APP_DIR) run .

test:
	$(GOENV) go -C $(APP_DIR) test ./...

fmt:
	$(GOENV) go -C $(APP_DIR) fmt ./...

compose-up: submodules
	$(COMPOSE) up --build

compose-down:
	$(COMPOSE) down

compose-logs:
	$(COMPOSE) logs -f

metrics-up: submodules
	$(COMPOSE) up --build dynamodb dynamodb-init metrics-service metrics-agent metrics-webview

metrics-down: compose-down

app-shell:
	$(COMPOSE) run --rm routing-slip-app sh
