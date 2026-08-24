package signal

import (
	"errors"
	"sync"
	"time"

	"github.com/wyw14/cry-111/internal/model"
)

type Selector struct {
	store    *Store
	mu       sync.Mutex
	commands map[string]model.SignalAspect
}

func NewSelector(store *Store) *Selector {
	return &Selector{store: store, commands: map[string]model.SignalAspect{}}
}

func (s *Selector) Command(id, routeID string, aspect model.SignalAspect, at time.Time) (model.SignalState, error) {
	if aspect == "" {
		return model.SignalState{}, errors.New("signal aspect is required")
	}
	state, err := s.store.Command(id, routeID, aspect, at)
	if err != nil {
		return model.SignalState{}, err
	}
	s.mu.Lock()
	s.commands[id] = aspect
	s.mu.Unlock()
	return state, nil
}

func (s *Selector) Observe(id string, selected, displayed model.SignalAspect, current int, at time.Time) (model.SignalState, error) {
	return s.store.UpdateCircuit(id, selected, displayed, current, at)
}

func (s *Selector) Requested(id string) (model.SignalAspect, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	aspect, ok := s.commands[id]
	return aspect, ok
}

func (s *Selector) Clear(id string) {
	s.mu.Lock()
	delete(s.commands, id)
	s.mu.Unlock()
}
