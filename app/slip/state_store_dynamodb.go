package slip

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const dynamoStateSK = "state"
const dynamoProcessingLockSK = "processing_lock"

// DynamoDBStateStore persists routing slip snapshots in DynamoDB.
//
// Expected table key schema:
//   - pk: string hash key
//   - sk: string range key
//
// The full snapshot is stored in the "snapshot" attribute as JSON. Additional
// top-level attributes make operational queries and dashboards easier.
type DynamoDBStateStore struct {
	client  *dynamodb.Client
	table   string
	ttlDays int
}

type dynamoProcessingLease struct {
	store     *DynamoDBStateStore
	messageID string
	owner     string
}

// NewDynamoDBStateStore creates a DynamoDB-backed state store.
func NewDynamoDBStateStore(client *dynamodb.Client, table string, ttlDays int) (*DynamoDBStateStore, error) {
	table = strings.TrimSpace(table)
	if table == "" {
		return nil, fmt.Errorf("dynamodb state table is required")
	}
	if client == nil {
		return nil, fmt.Errorf("dynamodb client is required")
	}
	return &DynamoDBStateStore{client: client, table: table, ttlDays: ttlDays}, nil
}

// TryAcquireProcessing claims a message_id with a conditional DynamoDB write.
// Expired claims can be replaced by another worker.
func (s *DynamoDBStateStore) TryAcquireProcessing(ctx context.Context, messageID, owner string, ttl time.Duration) (ProcessingLease, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return nil, false, fmt.Errorf("message_id is required for processing lock")
	}
	if ttl <= 0 {
		ttl = time.Minute
	}
	if owner == "" {
		owner = "dynamodb"
	}
	now := time.Now()
	expiresAt := now.Add(ttl).Unix()
	_, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.table),
		Item: map[string]types.AttributeValue{
			"pk":              &types.AttributeValueMemberS{Value: messageID},
			"sk":              &types.AttributeValueMemberS{Value: dynamoProcessingLockSK},
			"message_id":      &types.AttributeValueMemberS{Value: messageID},
			"owner":           &types.AttributeValueMemberS{Value: owner},
			"acquired_at":     &types.AttributeValueMemberS{Value: now.UTC().Format(time.RFC3339Nano)},
			"expires_at":      &types.AttributeValueMemberN{Value: strconv.FormatInt(expiresAt, 10)},
			"updated_at":      &types.AttributeValueMemberS{Value: now.UTC().Format(time.RFC3339Nano)},
			"updated_at_unix": &types.AttributeValueMemberN{Value: strconv.FormatInt(now.Unix(), 10)},
		},
		ConditionExpression: aws.String("attribute_not_exists(pk) OR expires_at < :now"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":now": &types.AttributeValueMemberN{Value: strconv.FormatInt(now.Unix(), 10)},
		},
	})
	if err != nil {
		var conditional *types.ConditionalCheckFailedException
		if errors.As(err, &conditional) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return dynamoProcessingLease{store: s, messageID: messageID, owner: owner}, true, nil
}

func (l dynamoProcessingLease) Release(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := l.store.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(l.store.table),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: l.messageID},
			"sk": &types.AttributeValueMemberS{Value: dynamoProcessingLockSK},
		},
		ConditionExpression: aws.String("owner = :owner"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":owner": &types.AttributeValueMemberS{Value: l.owner},
		},
	})
	if err != nil {
		var conditional *types.ConditionalCheckFailedException
		if errors.As(err, &conditional) {
			return nil
		}
	}
	return err
}

// Save writes or replaces the current snapshot.
func (s *DynamoDBStateStore) Save(ctx context.Context, snapshot MessageSnapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	item := map[string]types.AttributeValue{
		"pk":              &types.AttributeValueMemberS{Value: snapshot.ID},
		"sk":              &types.AttributeValueMemberS{Value: dynamoStateSK},
		"message_id":      &types.AttributeValueMemberS{Value: snapshot.ID},
		"workflow":        &types.AttributeValueMemberS{Value: snapshot.Workflow},
		"status":          &types.AttributeValueMemberS{Value: snapshot.Status},
		"cursor":          &types.AttributeValueMemberN{Value: strconv.Itoa(snapshot.Cursor)},
		"updated_at":      &types.AttributeValueMemberS{Value: snapshot.UpdatedAt.UTC().Format(time.RFC3339Nano)},
		"updated_at_unix": &types.AttributeValueMemberN{Value: strconv.FormatInt(snapshot.UpdatedAt.Unix(), 10)},
		"snapshot":        &types.AttributeValueMemberS{Value: string(data)},
	}
	if snapshot.CorrelationID != "" {
		item["correlation_id"] = &types.AttributeValueMemberS{Value: snapshot.CorrelationID}
	}
	if snapshot.TraceID != "" {
		item["trace_id"] = &types.AttributeValueMemberS{Value: snapshot.TraceID}
	}
	if s.ttlDays > 0 {
		expiresAt := snapshot.UpdatedAt.AddDate(0, 0, s.ttlDays).Unix()
		item["expires_at"] = &types.AttributeValueMemberN{Value: strconv.FormatInt(expiresAt, 10)}
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.table),
		Item:      item,
	})
	return err
}

// Load returns the latest snapshot for a message.
func (s *DynamoDBStateStore) Load(ctx context.Context, messageID string) (MessageSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return MessageSnapshot{}, err
	}
	output, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.table),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: messageID},
			"sk": &types.AttributeValueMemberS{Value: dynamoStateSK},
		},
	})
	if err != nil {
		return MessageSnapshot{}, err
	}
	if len(output.Item) == 0 {
		return MessageSnapshot{}, fmt.Errorf("%w: %s", ErrStateNotFound, messageID)
	}
	value, ok := output.Item["snapshot"].(*types.AttributeValueMemberS)
	if !ok || strings.TrimSpace(value.Value) == "" {
		return MessageSnapshot{}, fmt.Errorf("state for message %q has no snapshot payload", messageID)
	}
	var snapshot MessageSnapshot
	if err := json.Unmarshal([]byte(value.Value), &snapshot); err != nil {
		return MessageSnapshot{}, err
	}
	return snapshot, nil
}

// List scans stored snapshots and applies the filter in memory. It is intended
// for diagnostics and operational tooling, not hot-path processing.
func (s *DynamoDBStateStore) List(ctx context.Context, filter SnapshotFilter) ([]MessageSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	limit := int32(filter.Limit)
	if limit <= 0 {
		limit = 50
	}
	output, err := s.client.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(s.table),
		Limit:     aws.Int32(limit * 2),
	})
	if err != nil {
		return nil, err
	}
	out := make([]MessageSnapshot, 0, limit)
	for _, item := range output.Items {
		value, ok := item["snapshot"].(*types.AttributeValueMemberS)
		if !ok || strings.TrimSpace(value.Value) == "" {
			continue
		}
		var snapshot MessageSnapshot
		if err := json.Unmarshal([]byte(value.Value), &snapshot); err != nil {
			return nil, err
		}
		if snapshotMatches(snapshot, filter) {
			out = append(out, snapshot)
			if len(out) >= int(limit) {
				break
			}
		}
	}
	return out, nil
}
