// Package source loads framework configuration and workflows from local or AWS sources.
package source

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// Source loads serialized framework content.
type Source interface {
	Load(context.Context) ([]byte, error)
}

// Resolver resolves a referenced workflow relative to its parent source.
type Resolver interface {
	Resolve(string) (Source, error)
}

// Inline contains configuration directly in memory.
type Inline []byte

// Load returns inline content.
func (s Inline) Load(context.Context) ([]byte, error) { return []byte(s), nil }

// Local loads a local file and resolves child workflows relative to it.
type Local struct{ Path string }

// Load reads the local file.
func (s Local) Load(context.Context) ([]byte, error) { return os.ReadFile(s.Path) }

// Resolve locates a referenced local workflow.
func (s Local) Resolve(ref string) (Source, error) {
	if filepath.IsAbs(ref) {
		return Local{Path: ref}, nil
	}
	return Local{Path: filepath.Join(filepath.Dir(s.Path), ref)}, nil
}

// AWS identifies content stored in an AWS configuration service.
type AWS struct {
	Type           string
	Region         string
	Endpoint       string
	Bucket         string
	Key            string
	Name           string
	Table          string
	KeyAttribute   string
	ValueAttribute string
}

// Load retrieves content from S3, Secrets Manager, Parameter Store or DynamoDB.
func (s AWS) Load(ctx context.Context) ([]byte, error) {
	options := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(s.Region)}
	if s.Endpoint != "" {
		options = append(options, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("local", "local", "")))
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, options...)
	if err != nil {
		return nil, err
	}
	switch s.Type {
	case "s3":
		client := s3.NewFromConfig(cfg, endpointOption[s3.Options](s.Endpoint))
		out, err := client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.Bucket), Key: aws.String(s.Key)})
		if err != nil {
			return nil, err
		}
		defer out.Body.Close()
		return io.ReadAll(out.Body)
	case "secretsmanager", "secret":
		client := secretsmanager.NewFromConfig(cfg, endpointOption[secretsmanager.Options](s.Endpoint))
		out, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: aws.String(s.Name)})
		if err != nil {
			return nil, err
		}
		if out.SecretString != nil {
			return []byte(*out.SecretString), nil
		}
		return out.SecretBinary, nil
	case "ssm", "parameterstore":
		client := ssm.NewFromConfig(cfg, endpointOption[ssm.Options](s.Endpoint))
		out, err := client.GetParameter(ctx, &ssm.GetParameterInput{Name: aws.String(s.Name), WithDecryption: aws.Bool(true)})
		if err != nil || out.Parameter == nil || out.Parameter.Value == nil {
			return nil, err
		}
		return []byte(*out.Parameter.Value), nil
	case "dynamodb":
		client := dynamodb.NewFromConfig(cfg, endpointOption[dynamodb.Options](s.Endpoint))
		keyName := defaultString(s.KeyAttribute, "id")
		valueName := defaultString(s.ValueAttribute, "value")
		out, err := client.GetItem(ctx, &dynamodb.GetItemInput{
			TableName: aws.String(s.Table),
			Key:       map[string]types.AttributeValue{keyName: &types.AttributeValueMemberS{Value: s.Key}},
		})
		if err != nil {
			return nil, err
		}
		var item map[string]any
		if err := attributevalue.UnmarshalMap(out.Item, &item); err != nil {
			return nil, err
		}
		value, ok := item[valueName]
		if !ok {
			return nil, fmt.Errorf("attribute %q not found", valueName)
		}
		return []byte(fmt.Sprint(value)), nil
	default:
		return nil, fmt.Errorf("unsupported AWS source type %q", s.Type)
	}
}

// Resolve locates a referenced workflow using the same AWS source.
func (s AWS) Resolve(ref string) (Source, error) {
	switch s.Type {
	case "s3":
		s.Key = path.Join(path.Dir(s.Key), ref)
	case "secretsmanager", "secret", "ssm", "parameterstore":
		s.Name = path.Join(path.Dir(s.Name), ref)
	case "dynamodb":
		s.Key = path.Join(path.Dir(s.Key), ref)
	default:
		return nil, fmt.Errorf("unsupported AWS source type %q", s.Type)
	}
	return s, nil
}

func endpointOption[T any](endpoint string) func(*T) {
	return func(options *T) {
		if endpoint == "" {
			return
		}
		switch typed := any(options).(type) {
		case *s3.Options:
			typed.BaseEndpoint = aws.String(endpoint)
			typed.UsePathStyle = true
		case *secretsmanager.Options:
			typed.BaseEndpoint = aws.String(endpoint)
		case *ssm.Options:
			typed.BaseEndpoint = aws.String(endpoint)
		case *dynamodb.Options:
			typed.BaseEndpoint = aws.String(endpoint)
		}
	}
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
