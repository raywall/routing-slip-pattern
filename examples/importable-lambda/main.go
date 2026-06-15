package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	routing "github.com/raywall/routing-slip-pattern/app/framework"
	"github.com/raywall/routing-slip-pattern/app/source"
)

var runtime *routing.Runtime

func init() {
	var err error
	runtime, err = routing.New(context.Background(), routing.Options{
		ConfigSource:   source.Local{Path: "config.yaml"},
		WorkflowSource: source.Local{Path: "workflow.yaml"},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func handler(ctx context.Context, payload map[string]any) (any, error) {
	return runtime.Process(ctx, payload)
}

func main() {
	lambda.Start(handler)
}
