package main

import (
	"context"
	"log"

	routing "github.com/raywall/routing-slip-pattern/app/framework"
	"github.com/raywall/routing-slip-pattern/app/source"
)

func example(ctx context.Context) *routing.Runtime {
	runtime, err := routing.New(ctx, routing.Options{
		ConfigSource: source.AWS{
			Type: "s3", Region: "us-east-1", Bucket: "workflow-config", Key: "runtime/config.yaml",
		},
		WorkflowSource: source.AWS{
			Type: "dynamodb", Region: "us-east-1", Table: "workflows", Key: "product-processing",
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	return runtime
}
