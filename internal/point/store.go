package point

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/wyw14/cry-111/internal/model"
)

type Store struct {
	mu     sync.RWMutex
	points map[string]model.PointState
}

func NewStore(ids []string) *Store {
	points := make(map[string]model.PointState, len(ids))
	now := time.Now().UTC()
	for _, id := range ids {
		points[id] = model.PointState{ID: id, Commanded: model.PointNormal, Detected: model.PointNormal, Closed: true, ProofValid: true, DetectionSince: now, UpdatedAt: now}
	}
	return &Store{points: points}
}

func (s *Store) Command(id string, position model.PointPosition, at time.Time) (model.PointState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.points[id]
	if !ok {
		return model.PointState{}, errors.New("unknown point")
	}
	if state.LockedBy != "" && state.Commanded != position {
		return model.PointState{}, errors.New("point is route locked")
	}
	state.Commanded = position
	state.MotorRunning = true
	state.ProofValid = false
	state.Closed = false
	state.DetectionSince = time.Time{}
	state.UpdatedAt = at
	s.points[id] = state
	return state, nil
}

func (s *Store) UpdateDetection(id string, position model.PointPosition, closed, motorRunning bool, at time.Time) (model.PointState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.points[id]
	if !ok {
		return model.PointState{}, errors.New("unknown point")
	}
	changed := state.Detected != position || state.Closed != closed || state.MotorRunning != motorRunning
	state.Detected = position
	state.Closed = closed
	state.MotorRunning = motorRunning
	if changed {
		state.DetectionSince = at
		state.ProofValid = false
	}
	state.UpdatedAt = at
	s.points[id] = state
	return state, nil
}

func (s *Store) SetProof(id string, valid bool, at time.Time) (model.PointState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.points[id]
	if !ok {
		return model.PointState{}, errors.New("unknown point")
	}
	state.ProofValid = valid
	if valid {
		state.ProofRevision++
	}
	state.UpdatedAt = at
	s.points[id] = state
	return state, nil
}

func (s *Store) SetLock(id, routeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.points[id]
	if !ok {
		return errors.New("unknown point")
	}
	if state.LockedBy != "" && state.LockedBy != routeID {
		return errors.New("point already locked")
	}
	state.LockedBy = routeID
	s.points[id] = state
	return nil
}

func (s *Store) ClearLock(id, routeID string) {
	s.mu.Lock()
	state, ok := s.points[id]
	if ok && state.LockedBy == routeID {
		state.LockedBy = ""
		s.points[id] = state
	}
	s.mu.Unlock()
}

func (s *Store) Get(id string) (model.PointState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.points[id]
	return state, ok
}

func (s *Store) List() []model.PointState {
	s.mu.RLock()
	items := make([]model.PointState, 0, len(s.points))
	for _, item := range s.points {
		items = append(items, item)
	}
	s.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}
