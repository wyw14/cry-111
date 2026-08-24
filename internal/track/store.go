package track

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/wyw14/cry-111/internal/model"
)

type Store struct {
	mu       sync.RWMutex
	sections map[string]model.TrackState
	watchers []func(model.TrackState)
}

func NewStore(ids []string) *Store {
	sections := make(map[string]model.TrackState, len(ids))
	now := time.Now().UTC()
	for _, id := range ids {
		sections[id] = model.TrackState{ID: id, StableSince: now, LastTransition: now}
	}
	return &Store{sections: sections}
}

func (s *Store) RegisterWatcher(watcher func(model.TrackState)) {
	s.mu.Lock()
	s.watchers = append(s.watchers, watcher)
	s.mu.Unlock()
}

func (s *Store) SetOccupied(id string, occupied bool, direction string, at time.Time) (model.TrackState, error) {
	s.mu.Lock()
	current, ok := s.sections[id]
	if !ok {
		s.mu.Unlock()
		return model.TrackState{}, errors.New("unknown track section")
	}
	if current.Occupied != occupied {
		current.LastTransition = at
		current.StableSince = at
	}
	current.Occupied = occupied
	current.Direction = direction
	s.sections[id] = current
	watchers := append([]func(model.TrackState){}, s.watchers...)
	s.mu.Unlock()
	for _, watcher := range watchers {
		watcher(current)
	}
	return current, nil
}

func (s *Store) SetReservation(id, routeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.sections[id]
	if !ok {
		return errors.New("unknown track section")
	}
	current.ReservedBy = routeID
	s.sections[id] = current
	return nil
}

func (s *Store) ClearReservation(id, routeID string) {
	s.mu.Lock()
	current, ok := s.sections[id]
	if ok && current.ReservedBy == routeID {
		current.ReservedBy = ""
		s.sections[id] = current
	}
	s.mu.Unlock()
}

func (s *Store) Get(id string) (model.TrackState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.sections[id]
	return item, ok
}

func (s *Store) List() []model.TrackState {
	s.mu.RLock()
	items := make([]model.TrackState, 0, len(s.sections))
	for _, item := range s.sections {
		items = append(items, item)
	}
	s.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *Store) AllClear(ids []string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, id := range ids {
		item, ok := s.sections[id]
		if !ok || item.Occupied || item.ReservedBy != "" {
			return false
		}
	}
	return true
}
