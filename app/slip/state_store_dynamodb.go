package slip

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const dynamoStateSK = "state"

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
