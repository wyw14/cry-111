package approach

import (
	"errors"
	"sync"
	"time"
)

type Lock struct {
	RouteID   string        `json:"route_id"`
	SectionID string        `json:"section_id"`
	StartedAt time.Time     `json:"started_at"`
	Duration  time.Duration `json:"duration"`
	Released  bool          `json:"released"`
}

type LockingService struct {
	mu    sync.RWMutex
	locks map[string]Lock
}

func NewLockingService() *LockingService {
	return &LockingService{locks: map[string]Lock{}}
}

func (s *LockingService) Engage(routeID, sectionID string, at time.Time, duration time.Duration) (Lock, error) {
	if routeID == "" || sectionID == "" || duration <= 0 {
		return Lock{}, errors.New("route, section and positive duration are required")
	}
	lock := Lock{RouteID: routeID, SectionID: sectionID, StartedAt: at, Duration: duration}
	s.mu.Lock()
	s.locks[routeID] = lock
	s.mu.Unlock()
	return lock, nil
}

func (s *LockingService) Release(routeID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	lock, ok := s.locks[routeID]
	if !ok {
		return false
	}
	lock.Released = true
	s.locks[routeID] = lock
	return true
}

func (s *LockingService) Active(routeID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lock, ok := s.locks[routeID]
	return ok && !lock.Released
}

func (s *LockingService) Snapshot(routeID string) (Lock, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lock, ok := s.locks[routeID]
	return lock, ok
}
