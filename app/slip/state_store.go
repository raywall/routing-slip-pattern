package slip

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ErrStateNotFound indicates that no snapshot exists for a message.
var ErrStateNotFound = errors.New("routing-slip state not found")

// IsStateNotFound reports whether err means no snapshot exists.
func IsStateNotFound(err error) bool {
	return errors.Is(err, ErrStateNotFound)
}

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
		return MessageSnapshot{}, fmt.Errorf("%w: %s", ErrStateNotFound, messageID)
	}
	return snapshot, nil
}

// FileStateStore persists snapshots as JSON files. It is intentionally simple
// and useful for local development, tests and demos where DynamoDB is not
// available.
type FileStateStore struct {
	dir string
	mu  sync.Mutex
}

// NewFileStateStore creates a file-backed state store under dir.
func NewFileStateStore(dir string) (*FileStateStore, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = ".routing-slip-state"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &FileStateStore{dir: dir}, nil
}

// Save writes the snapshot atomically to disk.
func (s *FileStateStore) Save(ctx context.Context, snapshot MessageSnapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	path := s.path(snapshot.ID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Load reads the latest snapshot for a message from disk.
func (s *FileStateStore) Load(ctx context.Context, messageID string) (MessageSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return MessageSnapshot{}, err
	}
	data, err := os.ReadFile(s.path(messageID))
	if err != nil {
		if os.IsNotExist(err) {
			return MessageSnapshot{}, fmt.Errorf("%w: %s", ErrStateNotFound, messageID)
		}
		return MessageSnapshot{}, err
	}
	var snapshot MessageSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return MessageSnapshot{}, err
	}
	return snapshot, nil
}

func (s *FileStateStore) path(messageID string) string {
	return filepath.Join(s.dir, safeStateFileName(messageID)+".json")
}

func safeStateFileName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	var out strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			out.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			out.WriteRune(r)
		case r >= '0' && r <= '9':
			out.WriteRune(r)
		case r == '-', r == '_', r == '.':
			out.WriteRune(r)
		default:
			out.WriteByte('_')
		}
	}
	cleaned := strings.Trim(out.String(), "._-")
	if cleaned == "" {
		return "unknown"
	}
	return cleaned
}
