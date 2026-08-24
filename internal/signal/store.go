package signal

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/wyw14/cry-111/internal/model"
)

type Store struct {
	mu      sync.RWMutex
	signals map[string]model.SignalState
}

func NewStore(ids []string) *Store {
	items := make(map[string]model.SignalState, len(ids))
	now := time.Now().UTC()
	for _, id := range ids {
		items[id] = model.SignalState{ID: id, Commanded: model.AspectStop, Selected: model.AspectStop, Displayed: model.AspectStop, CurrentMilliAmps: 110, Proved: true, UpdatedAt: now}
	}
	return &Store{signals: items}
}

func (s *Store) Command(id, routeID string, aspect model.SignalAspect, at time.Time) (model.SignalState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.signals[id]
	if !ok {
		return model.SignalState{}, errors.New("unknown signal")
	}
	if aspect != model.AspectStop && aspect != model.AspectDark && state.RouteID != "" && state.RouteID != routeID {
		return model.SignalState{}, errors.New("signal belongs to another route")
	}
	state.Commanded = aspect
	state.RouteID = routeID
	state.Proved = false
	state.DarkProved = false
	state.UpdatedAt = at
	s.signals[id] = state
	return state, nil
}

func (s *Store) UpdateCircuit(id string, selected, displayed model.SignalAspect, current int, at time.Time) (model.SignalState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.signals[id]
	if !ok {
		return model.SignalState{}, errors.New("unknown signal")
	}
	state.Selected = selected
	state.Displayed = displayed
	state.CurrentMilliAmps = current
	state.UpdatedAt = at
	s.signals[id] = state
	return state, nil
}

func (s *Store) SetProof(id string, proved, dark bool, at time.Time) (model.SignalState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.signals[id]
	if !ok {
		return model.SignalState{}, errors.New("unknown signal")
	}
	state.Proved = proved
	state.DarkProved = dark
	if dark || state.Commanded == model.AspectStop {
		state.RouteID = ""
	}
	state.UpdatedAt = at
	s.signals[id] = state
	return state, nil
}

func (s *Store) Get(id string) (model.SignalState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.signals[id]
	return state, ok
}

func (s *Store) List() []model.SignalState {
	s.mu.RLock()
	items := make([]model.SignalState, 0, len(s.signals))
	for _, item := range s.signals {
		items = append(items, item)
	}
	s.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}
