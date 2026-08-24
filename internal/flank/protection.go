package flank

import (
	"errors"
	"sync"

	"github.com/wyw14/cry-111/internal/point"
)

type Protection struct {
	PointID       string `json:"point_id"`
	RouteID       string `json:"route_id"`
	ProofRevision uint64 `json:"proof_revision"`
}

type ProtectionService struct {
	reader *point.ProofReader
	locks  *point.LockManager
	mu     sync.RWMutex
	active map[string][]Protection
}

func NewProtectionService(reader *point.ProofReader, locks *point.LockManager) *ProtectionService {
	return &ProtectionService{reader: reader, locks: locks, active: map[string][]Protection{}}
}

func (s *ProtectionService) Establish(routeID string, pointIDs []string) ([]Protection, error) {
	proofs, err := s.reader.ReadAll(pointIDs)
	if err != nil {
		return nil, err
	}
	if err := s.locks.Lock(routeID, pointIDs); err != nil {
		return nil, err
	}
	protections := make([]Protection, 0, len(proofs))
	for _, proof := range proofs {
		protections = append(protections, Protection{PointID: proof.PointID, RouteID: routeID, ProofRevision: proof.Revision})
	}
	s.mu.Lock()
	s.active[routeID] = protections
	s.mu.Unlock()
	return append([]Protection(nil), protections...), nil
}

func (s *ProtectionService) Verify(routeID string) error {
	s.mu.RLock()
	items := append([]Protection(nil), s.active[routeID]...)
	s.mu.RUnlock()
	if len(items) == 0 {
		return errors.New("flank protection is not established")
	}
	for _, item := range items {
		proof, err := s.reader.Read(item.PointID)
		if err != nil || proof.Revision != item.ProofRevision {
			return errors.New("flank point proof changed after locking")
		}
	}
	return nil
}

func (s *ProtectionService) Release(routeID string) []string {
	s.mu.Lock()
	delete(s.active, routeID)
	s.mu.Unlock()
	return s.locks.Unlock(routeID)
}

func (s *ProtectionService) Snapshot(routeID string) []Protection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Protection(nil), s.active[routeID]...)
}
