package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/raywall/routing-slip-pattern/slip"
)

// AWSActionHandler executes controlled side effects against AWS services. It is
// intentionally explicit: each step states the service, action and target used
// to keep workflows traceable and resumable.
type AWSActionHandler struct{}

func (AWSActionHandler) Name() string { return "aws_action" }

func (h AWSActionHandler) Handle(ctx context.Context, msg *slip.Message, params map[string]any) (bool, error) {
	target := stringParam(params, "target", "aws_result")
	required := boolParam(params, "required", true)
	result, err := h.execute(ctx, msg, params)
	if err != nil {
		if required {
			return false, err
		}
		msg.Set(target+"_partial", true)
		msg.Set(target+"_error", err.Error())
		return true, nil
	}
	if target != "" {
		msg.Set(target, result)
	}
	msg.Set(target+"_executed_at", time.Now().Format(time.RFC3339))
	return true, nil
}

func (AWSActionHandler) execute(ctx context.Context, msg *slip.Message, params map[string]any) (map[string]any, error) {
	service := strings.ToLower(stringParam(params, "service", ""))
	action := strings.ToLower(stringParam(params, "action", ""))
	if service == "" || action == "" {
		return nil, fmt.Errorf("aws_action: service and action are required")
	}
	cfg, err := awsActionConfig(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("aws_action: load aws config: %w", err)
	}
	switch service {
	case "dynamodb", "dynamo":
		return executeDynamoDB(ctx, cfg, msg, action, params)
	case "s3":
		return executeS3(ctx, cfg, msg, action, params)
	case "sqs":
		return executeSQS(ctx, cfg, msg, action, params)
	case "sns":
		return executeSNS(ctx, cfg, msg, action, params)
	case "secretsmanager", "secrets_manager", "secret":
		return executeSecretsManager(ctx, cfg, msg, action, params)
	case "ssm", "parameter_store", "parameterstore":
		return executeSSM(ctx, cfg, msg, action, params)
	default:
		return nil, fmt.Errorf("aws_action: unsupported service %q", service)
	}
}

func awsActionConfig(ctx context.Context, params map[string]any) (aws.Config, error) {
	region := stringParam(params, "region", "us-east-1")
	endpoint := stringParam(params, "endpoint", "")
	opts := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(region)}
	if endpoint != "" {
		opts = append(opts, awsconfig.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: endpoint, SigningRegion: region}, nil
			}),
		))
	}
	return awsconfig.LoadDefaultConfig(ctx, opts...)
}

func executeDynamoDB(ctx context.Context, cfg aws.Config, msg *slip.Message, action string, params map[string]any) (map[string]any, error) {
	client := dynamodb.NewFromConfig(cfg)
	table := stringParam(params, "table", "")
	if table == "" {
		return nil, fmt.Errorf("dynamodb: table is required")
	}
	switch action {
	case "put", "create":
		item, err := dynamoAttributeMap(params, "item", msg)
		if err != nil {
			return nil, err
		}
		_, err = client.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(table), Item: item})
		return map[string]any{"service": "dynamodb", "action": "put", "table": table}, err
	case "get", "read":
		key, err := dynamoAttributeMap(params, "key", msg)
		if err != nil {
			return nil, err
		}
		out, err := client.GetItem(ctx, &dynamodb.GetItemInput{TableName: aws.String(table), Key: key})
		if err != nil {
			return nil, err
		}
		item := map[string]any{}
		if len(out.Item) > 0 {
			if err := attributevalue.UnmarshalMap(out.Item, &item); err != nil {
				return nil, err
			}
		}
		return map[string]any{"service": "dynamodb", "action": "get", "table": table, "item": item}, nil
	case "update":
		key, err := dynamoAttributeMap(params, "key", msg)
		if err != nil {
			return nil, err
		}
		expression := stringParam(params, "update_expression", "")
		if expression == "" {
			return nil, fmt.Errorf("dynamodb: update_expression is required")
		}
		values, err := optionalDynamoAttributeMap(params, "expression_attribute_values", msg)
		if err != nil {
			return nil, err
		}
		names := optionalStringMap(params, "expression_attribute_names", msg)
		input := &dynamodb.UpdateItemInput{
			TableName:                 aws.String(table),
			Key:                       key,
			UpdateExpression:          aws.String(expression),
			ExpressionAttributeNames:  names,
			ExpressionAttributeValues: values,
			ReturnValues:              dynamodbtypes.ReturnValue(stringParam(params, "return_values", "ALL_NEW")),
		}
		out, err := client.UpdateItem(ctx, input)
		if err != nil {
			return nil, err
		}
		attrs := map[string]any{}
		if len(out.Attributes) > 0 {
			if err := attributevalue.UnmarshalMap(out.Attributes, &attrs); err != nil {
				return nil, err
			}
		}
		return map[string]any{"service": "dynamodb", "action": "update", "table": table, "attributes": attrs}, nil
	case "delete", "remove":
		key, err := dynamoAttributeMap(params, "key", msg)
		if err != nil {
			return nil, err
		}
		_, err = client.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: aws.String(table), Key: key})
		return map[string]any{"service": "dynamodb", "action": "delete", "table": table}, err
	default:
		return nil, fmt.Errorf("dynamodb: unsupported action %q", action)
	}
}

func executeS3(ctx context.Context, cfg aws.Config, msg *slip.Message, action string, params map[string]any) (map[string]any, error) {
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if boolParam(params, "path_style", true) {
			o.UsePathStyle = true
		}
	})
	bucket := stringParam(params, "bucket", "")
	key := fmt.Sprintf("%v", interpolateAny(stringParam(params, "key", ""), msg))
	if bucket == "" || key == "" {
		return nil, fmt.Errorf("s3: bucket and key are required")
	}
	switch action {
	case "put", "create", "update":
		body, contentType, err := s3Body(params["body"], msg)
		if err != nil {
			return nil, err
		}
		_, err = client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(bucket),
			Key:         aws.String(key),
			Body:        bytes.NewReader(body),
			ContentType: aws.String(contentType),
		})
		return map[string]any{"service": "s3", "action": "put", "bucket": bucket, "key": key}, err
	case "get", "read":
		out, err := client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
		if err != nil {
			return nil, err
		}
		defer out.Body.Close()
		data, err := io.ReadAll(out.Body)
		if err != nil {
			return nil, err
		}
		return map[string]any{"service": "s3", "action": "get", "bucket": bucket, "key": key, "body": string(data)}, nil
	case "delete", "remove":
		_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
		return map[string]any{"service": "s3", "action": "delete", "bucket": bucket, "key": key}, err
	default:
		return nil, fmt.Errorf("s3: unsupported action %q", action)
	}
}

func executeSQS(ctx context.Context, cfg aws.Config, msg *slip.Message, action string, params map[string]any) (map[string]any, error) {
	if action != "send" && action != "publish" {
		return nil, fmt.Errorf("sqs: unsupported action %q", action)
	}
	queueURL := stringParam(params, "queue_url", "")
	if queueURL == "" {
		return nil, fmt.Errorf("sqs: queue_url is required")
	}
	body, err := messageBody(params["message"], msg)
	if err != nil {
		return nil, err
	}
	input := &sqs.SendMessageInput{QueueUrl: aws.String(queueURL), MessageBody: aws.String(body)}
	if delay := intParam(params["delay_seconds"]); delay > 0 {
		input.DelaySeconds = int32(delay)
	}
	out, err := sqs.NewFromConfig(cfg).SendMessage(ctx, input)
	if err != nil {
		return nil, err
	}
	return map[string]any{"service": "sqs", "action": "send", "queue_url": queueURL, "message_id": aws.ToString(out.MessageId)}, nil
}

func executeSNS(ctx context.Context, cfg aws.Config, msg *slip.Message, action string, params map[string]any) (map[string]any, error) {
	if action != "send" && action != "publish" {
		return nil, fmt.Errorf("sns: unsupported action %q", action)
	}
	topicARN := stringParam(params, "topic_arn", "")
	if topicARN == "" {
		return nil, fmt.Errorf("sns: topic_arn is required")
	}
	body, err := messageBody(params["message"], msg)
	if err != nil {
		return nil, err
	}
	input := &sns.PublishInput{TopicArn: aws.String(topicARN), Message: aws.String(body)}
	if subject := stringParam(params, "subject", ""); subject != "" {
		input.Subject = aws.String(subject)
	}
	out, err := sns.NewFromConfig(cfg).Publish(ctx, input)
	if err != nil {
		return nil, err
	}
	return map[string]any{"service": "sns", "action": "publish", "topic_arn": topicARN, "message_id": aws.ToString(out.MessageId)}, nil
}

func executeSecretsManager(ctx context.Context, cfg aws.Config, msg *slip.Message, action string, params map[string]any) (map[string]any, error) {
	client := secretsmanager.NewFromConfig(cfg)
	secretID := stringParam(params, "secret_id", stringParam(params, "name", ""))
	switch action {
	case "get", "read":
		if secretID == "" {
			return nil, fmt.Errorf("secretsmanager: secret_id is required")
		}
		out, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: aws.String(secretID)})
		if err != nil {
			return nil, err
		}
		return map[string]any{"service": "secretsmanager", "action": "get", "secret_id": secretID, "value": aws.ToString(out.SecretString)}, nil
	case "create":
		name := stringParam(params, "name", "")
		value := fmt.Sprintf("%v", interpolateAny(params["value"], msg))
		if name == "" {
			return nil, fmt.Errorf("secretsmanager: name is required")
		}
		out, err := client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{Name: aws.String(name), SecretString: aws.String(value)})
		if err != nil {
			return nil, err
		}
		return map[string]any{"service": "secretsmanager", "action": "create", "arn": aws.ToString(out.ARN), "name": name}, nil
	case "put", "update":
		if secretID == "" {
			return nil, fmt.Errorf("secretsmanager: secret_id is required")
		}
		value := fmt.Sprintf("%v", interpolateAny(params["value"], msg))
		out, err := client.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{SecretId: aws.String(secretID), SecretString: aws.String(value)})
		if err != nil {
			return nil, err
		}
		return map[string]any{"service": "secretsmanager", "action": "put", "arn": aws.ToString(out.ARN), "secret_id": secretID}, nil
	case "delete", "remove":
		if secretID == "" {
			return nil, fmt.Errorf("secretsmanager: secret_id is required")
		}
		input := &secretsmanager.DeleteSecretInput{SecretId: aws.String(secretID)}
		if boolParam(params, "force_delete_without_recovery", false) {
			input.ForceDeleteWithoutRecovery = aws.Bool(true)
		}
		out, err := client.DeleteSecret(ctx, input)
		if err != nil {
			return nil, err
		}
		return map[string]any{"service": "secretsmanager", "action": "delete", "arn": aws.ToString(out.ARN), "secret_id": secretID}, nil
	default:
		return nil, fmt.Errorf("secretsmanager: unsupported action %q", action)
	}
}

func executeSSM(ctx context.Context, cfg aws.Config, msg *slip.Message, action string, params map[string]any) (map[string]any, error) {
	client := ssm.NewFromConfig(cfg)
	name := stringParam(params, "name", "")
	if name == "" {
		return nil, fmt.Errorf("ssm: name is required")
	}
	switch action {
	case "get", "read":
		out, err := client.GetParameter(ctx, &ssm.GetParameterInput{Name: aws.String(name), WithDecryption: aws.Bool(boolParam(params, "with_decryption", true))})
		if err != nil {
			return nil, err
		}
		value := ""
		if out.Parameter != nil {
			value = aws.ToString(out.Parameter.Value)
		}
		return map[string]any{"service": "ssm", "action": "get", "name": name, "value": value}, nil
	case "put", "create", "update":
		value := fmt.Sprintf("%v", interpolateAny(params["value"], msg))
		parameterType := ssmtypes.ParameterType(stringParam(params, "type", "String"))
		out, err := client.PutParameter(ctx, &ssm.PutParameterInput{
			Name:      aws.String(name),
			Value:     aws.String(value),
			Type:      parameterType,
			Overwrite: aws.Bool(boolParam(params, "overwrite", action != "create")),
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"service": "ssm", "action": "put", "name": name, "version": out.Version}, nil
	case "delete", "remove":
		_, err := client.DeleteParameter(ctx, &ssm.DeleteParameterInput{Name: aws.String(name)})
		return map[string]any{"service": "ssm", "action": "delete", "name": name}, err
	default:
		return nil, fmt.Errorf("ssm: unsupported action %q", action)
	}
}

func dynamoAttributeMap(params map[string]any, key string, msg *slip.Message) (map[string]dynamodbtypes.AttributeValue, error) {
	raw, ok := params[key]
	if !ok {
		return nil, fmt.Errorf("dynamodb: %s is required", key)
	}
	value, ok := interpolateAny(raw, msg).(map[string]any)
	if !ok {
		return nil, fmt.Errorf("dynamodb: %s must be an object", key)
	}
	return attributevalue.MarshalMap(value)
}

func optionalDynamoAttributeMap(params map[string]any, key string, msg *slip.Message) (map[string]dynamodbtypes.AttributeValue, error) {
	if _, ok := params[key]; !ok {
		return nil, nil
	}
	return dynamoAttributeMap(params, key, msg)
}

func optionalStringMap(params map[string]any, key string, msg *slip.Message) map[string]string {
	raw, ok := params[key].(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		out[k] = fmt.Sprintf("%v", interpolateAny(v, msg))
	}
	return out
}

func s3Body(raw any, msg *slip.Message) ([]byte, string, error) {
	value := interpolateAny(raw, msg)
	switch typed := value.(type) {
	case string:
		return []byte(typed), "text/plain", nil
	case nil:
		return []byte{}, "application/octet-stream", nil
	default:
		data, err := json.Marshal(typed)
		return data, "application/json", err
	}
}

func messageBody(raw any, msg *slip.Message) (string, error) {
	value := interpolateAny(raw, msg)
	switch typed := value.(type) {
	case string:
		return typed, nil
	case nil:
		return "{}", nil
	default:
		data, err := json.Marshal(typed)
		return string(data), err
	}
}

func intParam(raw any) int {
	switch value := raw.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case json.Number:
		parsed, _ := strconv.Atoi(value.String())
		return parsed
	default:
		return 0
	}
}
