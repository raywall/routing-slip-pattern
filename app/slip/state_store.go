package slip

import (
	"context"
	"fmt"
	"sync"
)

// StateStore persists routing slip execution state by message ID.
type StateStore interface {
	Save(ctx context.Context, snapshot MessageSnapshot) error
	Load(ctx context.Context, messageID string) (MessageSnapshot, error)
}

// MemoryStateStore is useful for local tests and demos. Production adapters can
// implement the same interface with DynamoDB, PostgreSQL, Redis, or S3.
type MemoryStateStore struct {
	mu        sync.RWMutex
	snapshots map[string]MessageSnapshot
}

// NewMemoryStateStore creates an in-memory state store.
func NewMemoryStateStore() *MemoryStateStore {
	return &MemoryStateStore{
		snapshots: make(map[string]MessageSnapshot),
	}
}

// Save stores or replaces the snapshot for a message.
func (s *MemoryStateStore) Save(ctx context.Context, snapshot MessageSnapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshots[snapshot.ID] = snapshot
	return nil
}

// Load returns the latest snapshot for a message.
func (s *MemoryStateStore) Load(ctx context.Context, messageID string) (MessageSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return MessageSnapshot{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot, ok := s.snapshots[messageID]
	if !ok {
		return MessageSnapshot{}, fmt.Errorf("state for message %q not found", messageID)
	}
	return snapshot, nil
}
