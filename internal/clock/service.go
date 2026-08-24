package clock

import (
	"sync"
	"time"
)

type Service struct {
	mu       sync.RWMutex
	baseWall time.Time
	baseMono time.Time
	offset   time.Duration
	revision uint64
}

type Snapshot struct {
	WallTime time.Time     `json:"wall_time"`
	Elapsed  time.Duration `json:"elapsed"`
	Offset   time.Duration `json:"offset"`
	Revision uint64        `json:"revision"`
}

func New(now time.Time) *Service {
	return &Service{baseWall: now.UTC(), baseMono: time.Now()}
}

func (s *Service) WallNow() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.baseWall.Add(time.Since(s.baseMono)).Add(s.offset)
}

func (s *Service) MonotonicElapsed() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.baseMono)
}

func (s *Service) Correct(delta time.Duration) Snapshot {
	s.mu.Lock()
	s.offset += delta
	s.revision++
	snapshot := s.snapshotLocked()
	s.mu.Unlock()
	return snapshot
}

func (s *Service) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshotLocked()
}

func (s *Service) snapshotLocked() Snapshot {
	elapsed := time.Since(s.baseMono)
	return Snapshot{
		WallTime: s.baseWall.Add(elapsed).Add(s.offset),
		Elapsed:  elapsed,
		Offset:   s.offset,
		Revision: s.revision,
	}
}
