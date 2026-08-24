package journal

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/wyw14/cry-111/internal/model"
)

type Store struct {
	mu       sync.Mutex
	path     string
	sequence uint64
}

func NewStore(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("journal path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	store := &Store{path: path}
	events, err := store.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(events) > 0 {
		store.sequence = events[len(events)-1].Sequence
	}
	return store, nil
}

func (s *Store) Append(event model.Event) (model.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return model.Event{}, err
	}
	defer file.Close()
	s.sequence++
	event.Sequence = s.sequence
	encoded, err := json.Marshal(event)
	if err != nil {
		return model.Event{}, err
	}
	if _, err := file.Write(append(encoded, '\n')); err != nil {
		return model.Event{}, err
	}
	if err := file.Sync(); err != nil {
		return model.Event{}, err
	}
	return event.Clone(), nil
}

func (s *Store) ReadAll() ([]model.Event, error) {
	file, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return []model.Event{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	items := []model.Event{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event model.Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, err
		}
		items = append(items, event.Clone())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Store) Path() string {
	return s.path
}
