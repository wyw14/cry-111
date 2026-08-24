package point

import (
	"errors"
	"sync"
	"time"

	"github.com/wyw14/cry-111/internal/model"
)

type DetectionService struct {
	store       *Store
	stableDwell time.Duration
	mu          sync.Mutex
	pending     map[string]time.Time
}

func NewDetectionService(store *Store, stableDwell time.Duration) *DetectionService {
	return &DetectionService{store: store, stableDwell: stableDwell, pending: map[string]time.Time{}}
}

func (s *DetectionService) Observe(id string, position model.PointPosition, closed, motorRunning bool, at time.Time) (model.PointState, error) {
	state, err := s.store.UpdateDetection(id, position, closed, motorRunning, at)
	if err != nil {
		return model.PointState{}, err
	}
	s.mu.Lock()
	if motorRunning || !closed || position != state.Commanded {
		delete(s.pending, id)
		s.mu.Unlock()
		return s.store.SetProof(id, false, at)
	}
	started, exists := s.pending[id]
	if !exists {
		s.pending[id] = at
		s.mu.Unlock()
		return state, nil
	}
	stable := at.Sub(started) >= s.stableDwell
	if stable {
		delete(s.pending, id)
	}
	s.mu.Unlock()
	if stable {
		return s.store.SetProof(id, true, at)
	}
	return state, nil
}

func (s *DetectionService) RequireProof(id string) (model.PointState, error) {
	state, ok := s.store.Get(id)
	if !ok {
		return model.PointState{}, errors.New("unknown point")
	}
	if !state.PositionProved() {
		return state, errors.New("point position is not stably proved")
	}
	return state, nil
}

func (s *DetectionService) StableDwell() time.Duration {
	return s.stableDwell
}
