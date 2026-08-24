package route

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/wyw14/cry-111/internal/model"
)

type Store struct {
	mu     sync.RWMutex
	routes map[string]model.Route
	byName map[string]string
}

func NewStore() *Store {
	return &Store{routes: map[string]model.Route{}, byName: map[string]string{}}
}

func (s *Store) Create(name, kind, signalID string, at time.Time) (model.Route, error) {
	if name == "" || signalID == "" {
		return model.Route{}, errors.New("route name and signal are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if id := s.byName[name]; id != "" && s.routes[id].Active() {
		return model.Route{}, errors.New("an active route already uses this name")
	}
	route := model.Route{ID: model.NewIdentity().String(), Name: name, Kind: kind, SignalID: signalID, Phase: model.RouteRequested, UpdatedAt: at}
	s.routes[route.ID] = route
	s.byName[name] = route.ID
	return route.Clone(), nil
}

func (s *Store) Put(route model.Route) error {
	if route.ID == "" {
		return errors.New("route id is required")
	}
	s.mu.Lock()
	route.UpdatedAt = time.Now().UTC()
	s.routes[route.ID] = route.Clone()
	s.byName[route.Name] = route.ID
	s.mu.Unlock()
	return nil
}

func (s *Store) Get(id string) (model.Route, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	route, ok := s.routes[id]
	return route.Clone(), ok
}

func (s *Store) List() []model.Route {
	s.mu.RLock()
	items := make([]model.Route, 0, len(s.routes))
	for _, route := range s.routes {
		items = append(items, route.Clone())
	}
	s.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	return items
}

func (s *Store) Transition(id string, to model.RoutePhase, incident string) (model.Route, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	route, ok := s.routes[id]
	if !ok {
		return model.Route{}, errors.New("route not found")
	}
	if !canTransition(route.Phase, to) {
		return route.Clone(), errors.New("invalid route state transition")
	}
	route.Phase = to
	route.Incident = incident
	route.UpdatedAt = time.Now().UTC()
	s.routes[id] = route
	return route.Clone(), nil
}
