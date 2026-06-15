package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	metrics "github.com/raywall/custom-business-metrics/agent"
	routing "github.com/raywall/routing-slip-pattern/app/framework"
	"github.com/raywall/routing-slip-pattern/app/source"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	agent, err := metrics.New(metrics.Config{ServiceEndpoint: "http://localhost:8080/v1/metrics"})
	if err != nil {
		log.Fatal(err)
	}
	go func() { _ = agent.Run(ctx) }()
	defer agent.Close()

	runtime, err := routing.New(ctx, routing.Options{
		ConfigSource:   source.Local{Path: "config.yaml"},
		WorkflowSource: source.Local{Path: "workflow.yaml"},
		MetricsAgent:   agent,
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := runtime.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatal(err)
	}
}
