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
	"time"
)

// ErrStateNotFound indicates that no snapshot exists for a message.
var ErrStateNotFound = errors.New("routing-slip state not found")

// ErrProcessingLocked indicates that another worker is already processing the
// same message_id.
var ErrProcessingLocked = errors.New("routing-slip message is already being processed")

// IsStateNotFound reports whether err means no snapshot exists.
func IsStateNotFound(err error) bool {
	return errors.Is(err, ErrStateNotFound)
}

// IsProcessingLocked reports whether err means the message is currently claimed
// by another worker.
func IsProcessingLocked(err error) bool {
	return errors.Is(err, ErrProcessingLocked)
}

// StateStore persists routing slip execution state by message ID.
type StateStore interface {
	Save(ctx context.Context, snapshot MessageSnapshot) error
	Load(ctx context.Context, messageID string) (MessageSnapshot, error)
}

// ProcessingLease represents an acquired processing claim. Releasing the lease
// allows another worker to process the same message_id later.
type ProcessingLease interface {
	Release(ctx context.Context) error
}

// ProcessingLocker is implemented by stores that can prevent concurrent
// processing of the same message_id across workers.
type ProcessingLocker interface {
	TryAcquireProcessing(ctx context.Context, messageID, owner string, ttl time.Duration) (ProcessingLease, bool, error)
}

// SnapshotFilter filters stored snapshots for diagnostic tools.
type SnapshotFilter struct {
	MessageID     string
	CorrelationID string
	TraceID       string
	Workflow      string
	Status        string
	From          time.Time
	To            time.Time
	Limit         int
}

// StateSnapshotLister is optionally implemented by stores that can list states.
type StateSnapshotLister interface {
	List(ctx context.Context, filter SnapshotFilter) ([]MessageSnapshot, error)
}

// MemoryStateStore is useful for local tests and demos. Production adapters can
// implement the same interface with DynamoDB, PostgreSQL, Redis, or S3.
type MemoryStateStore struct {
	mu        sync.RWMutex
	snapshots map[string]MessageSnapshot
	locks     map[string]memoryProcessingLock
}

type memoryProcessingLock struct {
	owner     string
	expiresAt time.Time
}

type memoryProcessingLease struct {
	store     *MemoryStateStore
	messageID string
	owner     string
}

func (l memoryProcessingLease) Release(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	l.store.mu.Lock()
	defer l.store.mu.Unlock()
	if lock, ok := l.store.locks[l.messageID]; ok && lock.owner == l.owner {
		delete(l.store.locks, l.messageID)
	}
	return nil
}

// NewMemoryStateStore creates an in-memory state store.
func NewMemoryStateStore() *MemoryStateStore {
	return &MemoryStateStore{
		snapshots: make(map[string]MessageSnapshot),
		locks:     make(map[string]memoryProcessingLock),
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

// TryAcquireProcessing claims a message_id for one worker at a time.
func (s *MemoryStateStore) TryAcquireProcessing(ctx context.Context, messageID, owner string, ttl time.Duration) (ProcessingLease, bool, error) {
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
		owner = "memory"
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if lock, ok := s.locks[messageID]; ok && lock.expiresAt.After(now) {
		return nil, false, nil
	}
	s.locks[messageID] = memoryProcessingLock{owner: owner, expiresAt: now.Add(ttl)}
	return memoryProcessingLease{store: s, messageID: messageID, owner: owner}, true, nil
}

// List returns snapshots matching the filter.
func (s *MemoryStateStore) List(ctx context.Context, filter SnapshotFilter) ([]MessageSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]MessageSnapshot, 0, len(s.snapshots))
	for _, snapshot := range s.snapshots {
		if snapshotMatches(snapshot, filter) {
			out = append(out, snapshot)
			if filter.Limit > 0 && len(out) >= filter.Limit {
				break
			}
		}
	}
	return out, nil
}

// FileStateStore persists snapshots as JSON files. It is intentionally simple
// and useful for local development, tests and demos where DynamoDB is not
// available.
type FileStateStore struct {
	dir string
	mu  sync.Mutex
}

type fileProcessingLock struct {
	MessageID  string    `json:"message_id"`
	Owner      string    `json:"owner"`
	AcquiredAt time.Time `json:"acquired_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type fileProcessingLease struct {
	store     *FileStateStore
	messageID string
	owner     string
}

func (l fileProcessingLease) Release(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	l.store.mu.Lock()
	defer l.store.mu.Unlock()
	path := l.store.lockPath(l.messageID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var lock fileProcessingLock
	if json.Unmarshal(data, &lock) == nil && lock.Owner != l.owner {
		return nil
	}
	return os.Remove(path)
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

// List reads all JSON snapshots from disk and applies the filter.
func (s *FileStateStore) List(ctx context.Context, filter SnapshotFilter) ([]MessageSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	out := make([]MessageSnapshot, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var snapshot MessageSnapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			return nil, err
		}
		if snapshotMatches(snapshot, filter) {
			out = append(out, snapshot)
			if filter.Limit > 0 && len(out) >= filter.Limit {
				break
			}
		}
	}
	return out, nil
}

func (s *FileStateStore) path(messageID string) string {
	return filepath.Join(s.dir, safeStateFileName(messageID)+".json")
}

func (s *FileStateStore) lockPath(messageID string) string {
	return filepath.Join(s.dir, safeStateFileName(messageID)+".lock")
}

// TryAcquireProcessing claims a message_id using an atomic lock file. Stale
// locks are replaced after ttl.
func (s *FileStateStore) TryAcquireProcessing(ctx context.Context, messageID, owner string, ttl time.Duration) (ProcessingLease, bool, error) {
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
		owner = "file"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.lockPath(messageID)
	now := time.Now()
	if data, err := os.ReadFile(path); err == nil {
		var lock fileProcessingLock
		if json.Unmarshal(data, &lock) == nil && lock.ExpiresAt.After(now) {
			return nil, false, nil
		}
		_ = os.Remove(path)
	} else if err != nil && !os.IsNotExist(err) {
		return nil, false, err
	}

	lock := fileProcessingLock{
		MessageID:  messageID,
		Owner:      owner,
		AcquiredAt: now,
		ExpiresAt:  now.Add(ttl),
	}
	data, err := json.Marshal(lock)
	if err != nil {
		return nil, false, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, false, err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return nil, false, err
	}
	return fileProcessingLease{store: s, messageID: messageID, owner: owner}, true, nil
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

func snapshotMatches(snapshot MessageSnapshot, filter SnapshotFilter) bool {
	if filter.MessageID != "" && snapshot.ID != filter.MessageID {
		return false
	}
	if filter.CorrelationID != "" && snapshot.CorrelationID != filter.CorrelationID {
		return false
	}
	if filter.TraceID != "" && snapshot.TraceID != filter.TraceID {
		return false
	}
	if filter.Workflow != "" && snapshot.Workflow != filter.Workflow {
		return false
	}
	if filter.Status != "" && snapshot.Status != filter.Status {
		return false
	}
	if !filter.From.IsZero() && snapshot.UpdatedAt.Before(filter.From) {
		return false
	}
	if !filter.To.IsZero() && snapshot.UpdatedAt.After(filter.To) {
		return false
	}
	return true
}
