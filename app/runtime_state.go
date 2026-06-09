package main

import (
	"context"
	"fmt"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/raywall/routing-slip-pattern/app/slip"
)

func buildStateStore(ctx context.Context, cfg *AppConfig) (slip.StateStore, error) {
	if !cfg.Features.PersistentStateEnabled {
		return slip.NewMemoryStateStore(), nil
	}

	switch strings.ToLower(strings.TrimSpace(cfg.StateStore.Type)) {
	case "", "memory":
		return slip.NewMemoryStateStore(), nil
	case "file":
		return slip.NewFileStateStore(cfg.StateStore.Path)
	case "dynamodb":
		return buildDynamoDBStateStore(ctx, cfg.StateStore)
	default:
		return nil, fmt.Errorf("unsupported state store type %q", cfg.StateStore.Type)
	}
}

func buildDynamoDBStateStore(ctx context.Context, cfg StateStoreConfig) (slip.StateStore, error) {
	options := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}
	if strings.TrimSpace(cfg.Endpoint) != "" {
		options = append(options,
			awsconfig.WithBaseEndpoint(cfg.Endpoint),
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("local", "local", "")),
		)
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, options...)
	if err != nil {
		return nil, err
	}
	client := dynamodb.NewFromConfig(awsCfg)
	return slip.NewDynamoDBStateStore(client, cfg.Table, cfg.TTLDays)
}

func stateOptions(cfg *AppConfig, workflow *WorkflowConfig) slip.StateOptions {
	return slip.StateOptions{
		Workflow:               workflow.Name,
		WorkflowVersion:        workflow.Version,
		IdempotencyEnabled:     cfg.StateStore.Idempotency.Enabled,
		IdempotencyKeyTemplate: cfg.StateStore.Idempotency.KeyTemplate,
	}
}
