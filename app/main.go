package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/raywall/routing-slip-pattern/config"
	"github.com/raywall/routing-slip-pattern/handlers"
	"github.com/raywall/routing-slip-pattern/slip"
)

// ---------------------------------------------------------------------------
// Bootstrap: build a Router and register all available handlers.
// ---------------------------------------------------------------------------

func buildRouter(logger *slog.Logger, policy slip.ErrorPolicy) *slip.Router {
	return buildRouterWithOptions(logger, policy)
}

func buildRouterWithOptions(logger *slog.Logger, policy slip.ErrorPolicy, opts ...slip.RouterOption) *slip.Router {
	routerOpts := []slip.RouterOption{
		slip.WithLogger(logger),
		slip.WithErrorPolicy(policy),
		slip.WithMiddleware(
			slip.RecoveryMiddleware(),
			slip.LoggingMiddleware(logger),
		),
	}
	if envBool("ROUTING_SLIP_TRACING_ENABLED", true) {
		routerOpts = append(routerOpts, slip.WithMiddleware(slip.TracingMiddleware()))
	}
	routerOpts = append(routerOpts, opts...)

	r := slip.NewRouter(routerOpts...)

	r.MustRegister(handlers.ValidationHandler{})
	r.MustRegister(handlers.EnrichmentHandler{})
	r.MustRegister(handlers.AssertHandler{})
	r.MustRegister(handlers.ComputeHandler{})
	r.MustRegister(handlers.CELHandler{})
	r.MustRegister(handlers.FilterArrayHandler{})
	r.MustRegister(handlers.ArrayTransformHandler{})
	r.MustRegister(handlers.TransformHandler{})
	r.MustRegister(handlers.ConditionGate{})
	r.MustRegister(handlers.JumpIfHandler{})
	r.MustRegister(&handlers.NotificationHandler{}) // pointer because Send field may be set
	r.MustRegister(handlers.LogHandler{})
	r.MustRegister(handlers.AuditHandler{})
	r.MustRegister(handlers.DatadogMetricHandler{})
	r.MustRegister(handlers.GraphQLEnrichmentHandler{DefaultEndpoint: env("GRAPHQL_ENDPOINT", "http://localhost:8090/graphql")})
	r.MustRegister(handlers.RESTCallHandler{})
	r.MustRegister(handlers.AWSActionHandler{})

	return r
}

func buildScenarioRouter(logger *slog.Logger, workflow string, store slip.StateStore, flaky *FlakyHandler) *slip.Router {
	metricsEndpoint := env("METRICS_ENDPOINT", "http://localhost:8080/v1/metrics")
	router := buildRouterWithOptions(logger, slip.StopOnError,
		slip.WithStateStore(store),
		slip.WithMiddleware(slip.MetricsMiddleware(
			slip.HTTPMetricsEmitter{Endpoint: metricsEndpoint},
			slip.MetricsOptions{
				Workflow: workflow,
				Segment:  "local-demo",
				Source:   "routing-slip-app",
				Tags: map[string]string{
					"run_id": env("RUN_ID", time.Now().Format("20060102150405")),
				},
			},
			logger,
		)),
	)
	if flaky != nil {
		router.MustRegister(flaky)
	}
	return router
}

// ---------------------------------------------------------------------------
// Demo 1 – Fluent API (hard-coded slip in Go code)
// ---------------------------------------------------------------------------

func demoFluentAPI(router *slip.Router) {
	banner("Demo 1 – Fluent API")

	msg := slip.NewMessage("MSG-001", map[string]any{
		"customer_id": "cust-42",
		"product_id":  "SKU-9000",
		"quantity":    3,
	})

	steps := slip.NewSlip().
		Step("validate", map[string]any{
			"required": []string{"customer_id", "product_id", "quantity"},
		}).
		Step("enrich", map[string]any{
			"prefix": "order_",
			"data": map[string]any{
				"status": "PENDING",
				"source": "fluent-demo",
			},
		}).
		Step("transform", map[string]any{
			"field":     "customer_id",
			"operation": "uppercase",
		}).
		Step("notify", map[string]any{
			"channel":   "slack",
			"recipient": "#orders",
			"template":  "Order from {customer_id}: {product_id} x{quantity}",
		}).
		Step("audit", map[string]any{
			"event":  "demo1.complete",
			"fields": []string{"customer_id", "order_status"},
		}).
		Build()

	msg.AttachSlip(steps)

	if err := router.Process(context.Background(), msg); err != nil {
		fmt.Printf("  ❌ Error: %v\n", err)
	} else {
		printResult(msg)
	}
}

// ---------------------------------------------------------------------------
// Demo 2 – JSON-driven workflow
// ---------------------------------------------------------------------------

func demoJSONWorkflow(router *slip.Router) {
	banner("Demo 2 – JSON workflow config")

	cfg, err := config.LoadFromFile("examples/order_workflow.json")
	if err != nil {
		fmt.Printf("  ❌ Failed to load config: %v\n", err)
		return
	}

	msg := slip.NewMessage("MSG-002", map[string]any{
		"customer_id": "cust-007",
		"product_id":  "WIDGET-X",
		"quantity":    10,
	})
	msg.AttachSlip(cfg.ToSlip())

	if err := router.Process(context.Background(), msg); err != nil {
		fmt.Printf("  ❌ Error: %v\n", err)
	} else {
		printResult(msg)
	}
}

// ---------------------------------------------------------------------------
// Demo 3 – Dynamic slip with condition gate (early exit)
// ---------------------------------------------------------------------------

func demoDynamicSlip(router *slip.Router) {
	banner("Demo 3 – Condition gate (early exit)")

	msg := slip.NewMessage("MSG-003", map[string]any{
		"customer_id": "cust-banned",
		"product_id":  "SKU-1",
		"quantity":    1,
		"status":      "BLOCKED", // gate will stop here
	})

	steps := slip.NewSlip().
		Step("validate", map[string]any{
			"required": []string{"customer_id", "product_id"},
		}).
		Step("condition", map[string]any{
			"field":  "status",
			"equals": "ACTIVE", // message has "BLOCKED" → will stop
		}).
		Step("notify", map[string]any{ // should NOT execute
			"channel":   "email",
			"recipient": "team@example.com",
			"template":  "Should never arrive",
		}).
		Build()

	msg.AttachSlip(steps)

	if err := router.Process(context.Background(), msg); err != nil {
		fmt.Printf("  ❌ Error: %v\n", err)
	} else {
		stopped, _ := msg.Get("gate_stopped")
		fmt.Printf("  Gate stopped early: %v\n", stopped)
		printResult(msg)
	}
}

// ---------------------------------------------------------------------------
// Demo 4 – ContinueOnError policy
// ---------------------------------------------------------------------------

func demoContinueOnError() {
	banner("Demo 4 – ContinueOnError policy")

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	router := buildRouter(logger, slip.ContinueOnError)

	msg := slip.NewMessage("MSG-004", map[string]any{
		"product_id": "SKU-777",
		// customer_id is deliberately missing to trigger validation error
	})

	steps := slip.NewSlip().
		Step("validate", map[string]any{
			"required":        []string{"customer_id", "product_id"},
			"stop_on_failure": false, // log only
		}).
		Step("enrich", map[string]any{
			"data": map[string]any{"fallback": true},
		}).
		Step("audit", map[string]any{
			"event":  "demo4.fallback",
			"fields": []string{"product_id", "fallback"},
		}).
		Build()

	msg.AttachSlip(steps)

	if err := router.Process(context.Background(), msg); err != nil {
		fmt.Printf("  ❌ Unexpected error: %v\n", err)
	} else {
		printResult(msg)
	}
}

// ---------------------------------------------------------------------------
// Demo 5 – Context cancellation
// ---------------------------------------------------------------------------

func demoContextCancel(router *slip.Router) {
	banner("Demo 5 – Context cancellation")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	msg := slip.NewMessage("MSG-005", map[string]any{"customer_id": "cust-1"})
	msg.AttachSlip(slip.NewSlip().
		Step("enrich", map[string]any{"data": map[string]any{"x": 1}}).
		Step("audit", map[string]any{"event": "demo5"}).
		Build(),
	)

	err := router.Process(ctx, msg)
	fmt.Printf("  Context cancelled as expected: %v\n", err != nil)
}

// ---------------------------------------------------------------------------
// Demo 6 - Resumable processing from the failed step
// ---------------------------------------------------------------------------

type FlakyHandler struct {
	failuresLeft int
}

func (f *FlakyHandler) Name() string { return "flaky" }

func (f *FlakyHandler) Handle(ctx context.Context, msg *slip.Message, params map[string]any) (bool, error) {
	if f.failuresLeft > 0 {
		f.failuresLeft--
		msg.Set("flaky_attempted", true)
		return false, fmt.Errorf("temporary dependency unavailable")
	}
	msg.Set("flaky_recovered", true)
	return true, nil
}

func demoResumableProcessing(logger *slog.Logger) {
	banner("Demo 6 - Resumable processing")

	store := slip.NewMemoryStateStore()
	flaky := &FlakyHandler{failuresLeft: 1}
	router := buildRouterWithOptions(logger, slip.StopOnError, slip.WithStateStore(store))
	router.MustRegister(flaky)

	msg := slip.NewMessage("MSG-006", map[string]any{
		"customer_id": "cust-resume",
		"product_id":  "SKU-RESUME",
		"quantity":    1,
	})
	msg.AttachSlip(slip.NewSlip().
		Step("validate", map[string]any{"required": []string{"customer_id", "product_id"}}).
		Step("flaky").
		Step("enrich", map[string]any{"data": map[string]any{"resumed": true}}).
		Step("audit", map[string]any{"event": "demo6.resumed"}).
		Build(),
	)

	firstErr := router.Process(context.Background(), msg)
	fmt.Printf("  First run stopped with error: %v\n", firstErr != nil)

	snapshot, err := store.Load(context.Background(), msg.ID)
	if err != nil {
		fmt.Printf("  Failed to load snapshot: %v\n", err)
		return
	}
	fmt.Printf("  Persisted cursor after failure: %d\n", snapshot.Cursor)

	resumed := slip.MessageFromSnapshot(snapshot)
	secondErr := router.Process(context.Background(), resumed)
	fmt.Printf("  Resume completed with error: %v\n", secondErr != nil)
	printResult(resumed)
}

// ---------------------------------------------------------------------------
// Scenario suite - used by make run after make prepare
// ---------------------------------------------------------------------------

type WorkflowScenario struct {
	Name        string
	MessageID   string
	Description string
	Payload     map[string]any
	Steps       []slip.StepDef
	Resume      bool
	Flaky       *FlakyHandler
}

func runWorkflowScenarios(logger *slog.Logger) {
	banner("Routing Slip Scenario Suite")
	fmt.Printf("  Metrics dashboard: http://localhost:5173\n")
	fmt.Printf("  Metrics endpoint:  %s\n", env("METRICS_ENDPOINT", "http://localhost:8080/v1/metrics"))
	fmt.Printf("  GraphQL endpoint:  %s\n", env("GRAPHQL_ENDPOINT", "http://localhost:8090/graphql"))
	fmt.Printf("  External API:       %s\n", env("EXTERNAL_API_URL", "http://localhost:8091"))

	scenarios := []WorkflowScenario{
		{
			Name:        "payment-event-fulfillment",
			MessageID:   "PAYMENT-EVENT-001",
			Description: "Evento de pagamento aprovado consulta pedido, emite nota fiscal, aciona expedicao e baixa estoque",
			Payload: map[string]any{
				"evento": "PAGAMENTO_APROVADO",
				"payload": map[string]any{
					"pagamento_id": "PAG-5544",
					"pedido_id":    "PED-9988",
					"valor_pago":   150.0,
				},
				"correlation_id": newCorrelationUUID(),
				"received_at":    "2026-05-13T21:55:00Z",
			},
			Steps: paymentFulfillmentSteps(),
		},
		{
			Name:        "order-ok",
			MessageID:   "SCN-OK-001",
			Description: "Processamento completo com enriquecimento GraphQL",
			Payload: map[string]any{
				"customer_id":    "cust-42",
				"product_id":     "SKU-9000",
				"quantity":       3,
				"correlation_id": newCorrelationUUID(),
			},
			Steps: standardEnrichedSteps(),
		},
		{
			Name:        "order-stopped-by-decision",
			MessageID:   "SCN-STOP-001",
			Description: "Processamento represado por decisão funcional após enriquecimento",
			Payload: map[string]any{
				"customer_id":    "cust-blocked",
				"product_id":     "SKU-LOCK",
				"quantity":       1,
				"correlation_id": newCorrelationUUID(),
			},
			Steps: standardEnrichedSteps(),
		},
		{
			Name:        "order-fail-and-resume",
			MessageID:   "SCN-RESUME-001",
			Description: "Falha técnica no meio do fluxo e reprocessamento a partir do cursor salvo",
			Payload: map[string]any{
				"customer_id":    "cust-resume",
				"product_id":     "SKU-RETRY",
				"quantity":       2,
				"correlation_id": newCorrelationUUID(),
			},
			Steps:  resumableSteps(),
			Resume: true,
			Flaky:  &FlakyHandler{failuresLeft: 1},
		},
	}

	for _, scenario := range scenarios {
		runScenario(logger, scenario)
	}
}

func runScenario(logger *slog.Logger, scenario WorkflowScenario) {
	banner("Scenario - " + scenario.Name)
	fmt.Printf("  %s\n", scenario.Description)

	store := slip.NewMemoryStateStore()
	router := buildScenarioRouter(logger, scenario.Name, store, scenario.Flaky)
	msg := slip.NewMessage(scenario.MessageID, scenario.Payload)
	msg.AttachSlip(scenario.Steps)

	err := router.Process(context.Background(), msg)
	if err != nil {
		fmt.Printf("  First run error: %v\n", err)
	} else {
		fmt.Println("  First run completed")
	}

	if scenario.Resume && err != nil {
		snapshot, loadErr := store.Load(context.Background(), msg.ID)
		if loadErr != nil {
			fmt.Printf("  Snapshot load failed: %v\n", loadErr)
			return
		}
		fmt.Printf("  Resuming from cursor %d (%s)\n", snapshot.Cursor, stepName(snapshot))
		resumed := slip.MessageFromSnapshot(snapshot)
		err = router.Process(context.Background(), resumed)
		if err != nil {
			fmt.Printf("  Resume error: %v\n", err)
		} else {
			fmt.Println("  Resume completed")
		}
		printResult(resumed)
		return
	}

	printResult(msg)
}

func paymentFulfillmentSteps() []slip.StepDef {
	externalAPI := env("EXTERNAL_API_URL", "http://localhost:8091")
	externalAPISerial := env("EXTERNAL_API_SERIAL", "b7af3a9e-6d1a-4b15-9837-3e0f0b47e5b4")

	return slip.NewSlip().
		Step("validate", map[string]any{
			"required": []string{"evento", "payload.pagamento_id", "payload.pedido_id", "payload.valor_pago"},
		}).
		Step("condition", map[string]any{
			"field":  "evento",
			"equals": "PAGAMENTO_APROVADO",
		}).
		Step("graphql_enrich", map[string]any{
			"query":       "query ($pedidoID: String!) { dataSources(pedidoID: $pedidoID) { order { pedido_id cliente_id status valor_total endereco_entrega itens { produto_id quantidade } } } }",
			"variables":   map[string]any{"pedidoID": "{payload.pedido_id}"},
			"target":      "pedido",
			"result_path": "dataSources.order",
			"timeout_ms":  1500,
			"required":    true,
		}).
		Step("rest_call", map[string]any{
			"base_url": externalAPI,
			"method":   "POST",
			"endpoint": "/lambda/notas-fiscais",
			"target":   "nota_fiscal",
			"headers": map[string]any{
				"X-Serial-Number": externalAPISerial,
			},
			"body": map[string]any{
				"pedido_id":    "{pedido.pedido_id}",
				"cliente_id":   "{pedido.cliente_id}",
				"valor_total":  "{pedido.valor_total}",
				"itens":        "{pedido.itens}",
				"pagamento_id": "{payload.pagamento_id}",
			},
			"required": true,
		}).
		Step("condition", map[string]any{
			"field":  "nota_fiscal.status",
			"equals": "EMITIDA",
		}).
		Step("rest_call", map[string]any{
			"base_url": externalAPI,
			"method":   "POST",
			"endpoint": "/api/expedicao",
			"target":   "expedicao",
			"headers": map[string]any{
				"X-Serial-Number": externalAPISerial,
			},
			"body": map[string]any{
				"pedido_id":        "{pedido.pedido_id}",
				"cliente_id":       "{pedido.cliente_id}",
				"endereco_entrega": "{pedido.endereco_entrega}",
				"itens":            "{pedido.itens}",
				"nota_fiscal_id":   "{nota_fiscal.nota_fiscal_id}",
			},
			"required": true,
		}).
		Step("rest_call", map[string]any{
			"base_url": externalAPI,
			"method":   "PUT",
			"endpoint": "/api/estoque/baixar",
			"target":   "estoque_baixa",
			"headers": map[string]any{
				"X-Serial-Number": externalAPISerial,
			},
			"body": map[string]any{
				"pedido_id": "{pedido.pedido_id}",
				"itens":     "{pedido.itens}",
			},
			"required": true,
		}).
		Step("audit", map[string]any{
			"event": "payment.fulfillment.completed",
			"fields": []string{
				"evento",
				"payload.pedido_id",
				"pedido.status",
				"nota_fiscal.status",
				"expedicao.codigo_rastreio",
				"estoque_baixa.status",
			},
		}).
		Build()
}

func standardEnrichedSteps() []slip.StepDef {
	return slip.NewSlip().
		Step("validate", map[string]any{
			"required": []string{"customer_id", "product_id", "quantity"},
		}).
		Step("graphql_enrich", map[string]any{
			"query":       "query ($customerID: String!) { dataSources(customerID: $customerID) { customer { id status riskSegment creditLimit sourceSystem } } }",
			"variables":   map[string]any{"customerID": "{customer_id}"},
			"target":      "customer_profile",
			"result_path": "dataSources.customer",
			"timeout_ms":  1500,
			"required":    true,
		}).
		Step("condition", map[string]any{
			"field":  "customer_profile.status",
			"equals": "ACTIVE",
		}).
		Step("transform", map[string]any{
			"field":     "customer_id",
			"operation": "uppercase",
			"target":    "customer_id",
		}).
		Step("enrich", map[string]any{
			"prefix": "order_",
			"data": map[string]any{
				"status": "READY",
				"source": "routing-slip-suite",
			},
		}).
		Step("audit", map[string]any{
			"event":  "scenario.processed",
			"fields": []string{"customer_id", "product_id", "customer_profile.status", "order_status"},
		}).
		Build()
}

func resumableSteps() []slip.StepDef {
	steps := standardEnrichedSteps()
	res := make([]slip.StepDef, 0, len(steps)+1)
	res = append(res, steps[:2]...)
	res = append(res, slip.StepDef{Name: "flaky", Params: map[string]any{}})
	res = append(res, steps[2:]...)
	return res
}

func stepName(snapshot slip.MessageSnapshot) string {
	if snapshot.Cursor < 0 || snapshot.Cursor >= len(snapshot.Slip) {
		return "end"
	}
	return snapshot.Slip[snapshot.Cursor].Name
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func banner(title string) {
	fmt.Printf("\n%s\n%s\n", title, repeat("─", len(title)))
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

func printResult(msg *slip.Message) {
	payload, _ := json.MarshalIndent(msg.Payload, "  ", "  ")
	fmt.Printf("  Steps executed: %d\n", len(msg.History))
	fmt.Printf("  Errors: %d\n", len(msg.Errors))
	fmt.Printf("  Payload:\n  %s\n", payload)

	fmt.Println("  History:")
	for _, h := range msg.History {
		skip := ""
		if h.Skipped {
			skip = " (skipped)"
		}
		fmt.Printf("    • %-16s %v%s\n", h.Step, h.Duration.Round(time.Microsecond), skip)
	}
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	configPath := flag.String("config", "", "path to config.yaml")
	workflowPath := flag.String("workflow", "", "path to workflow yaml")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	if strings.TrimSpace(*configPath) != "" {
		if strings.TrimSpace(*workflowPath) == "" {
			logger.Error("workflow config is required when --config is used", slog.String("flag", "--workflow"))
			os.Exit(1)
		}
		cfg, err := loadAppConfig(*configPath)
		if err != nil {
			logger.Error("failed to load config", slog.String("error", err.Error()))
			os.Exit(1)
		}
		workflow, err := loadWorkflowConfig(*workflowPath)
		if err != nil {
			logger.Error("failed to load workflow", slog.String("error", err.Error()))
			os.Exit(1)
		}
		if err := runConfiguredApp(context.Background(), cfg, workflow, logger); err != nil {
			logger.Error("configured workflow stopped", slog.String("error", err.Error()))
			os.Exit(1)
		}
		return
	}

	runWorkflowScenarios(logger)
	fmt.Println("\n✅ Scenario suite complete.")
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}
