package journal

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Snapshot struct {
	Revision  uint64          `json:"revision"`
	CreatedAt time.Time       `json:"created_at"`
	Payload   json.RawMessage `json:"payload"`
}

type SnapshotStore struct {
	mu   sync.Mutex
	path string
}

func NewSnapshotStore(path string) (*SnapshotStore, error) {
	if path == "" {
		return nil, errors.New("snapshot path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return &SnapshotStore{path: path}, nil
}

func (s *SnapshotStore) Save(revision uint64, value any, at time.Time) (Snapshot, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return Snapshot{}, err
	}
	snapshot := Snapshot{Revision: revision, CreatedAt: at, Payload: payload}
	encoded, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return Snapshot{}, err
	}
	temporary := s.path + ".next"
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.WriteFile(temporary, append(encoded, '\n'), 0o644); err != nil {
		return Snapshot{}, err
	}
	if err := os.Rename(temporary, s.path); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

func (s *SnapshotStore) Load() (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		return Snapshot{}, err
	}
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

func (s *SnapshotStore) Path() string {
	return s.path
}
