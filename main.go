package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/raywall/routing-slip-pattern/config"
	"github.com/raywall/routing-slip-pattern/handlers"
	"github.com/raywall/routing-slip-pattern/slip"
)

// ---------------------------------------------------------------------------
// Bootstrap: build a Router and register all available handlers.
// ---------------------------------------------------------------------------

func buildRouter(logger *slog.Logger, policy slip.ErrorPolicy) *slip.Router {
	r := slip.NewRouter(
		slip.WithLogger(logger),
		slip.WithErrorPolicy(policy),
		slip.WithMiddleware(
			slip.RecoveryMiddleware(),
			slip.LoggingMiddleware(logger),
		),
	)

	r.MustRegister(handlers.ValidationHandler{})
	r.MustRegister(handlers.EnrichmentHandler{})
	r.MustRegister(handlers.TransformHandler{})
	r.MustRegister(handlers.ConditionGate{})
	r.MustRegister(&handlers.NotificationHandler{}) // pointer because Send field may be set
	r.MustRegister(handlers.AuditHandler{})

	return r
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
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	router := buildRouter(logger, slip.StopOnError)

	demoFluentAPI(router)
	demoJSONWorkflow(router)
	demoDynamicSlip(router)
	demoContinueOnError()
	demoContextCancel(router)

	fmt.Println("\n✅ All demos complete.")
}
