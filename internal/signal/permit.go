package signal

import (
	"errors"
	"sync"

	"github.com/wyw14/cry-111/internal/model"
)

type PermitSnapshot struct {
	RouteID           string `json:"route_id"`
	MainLocked        bool   `json:"main_locked"`
	FlankLocked       bool   `json:"flank_locked"`
	OverlapLocked     bool   `json:"overlap_locked"`
	CrossingProtected bool   `json:"crossing_protected"`
	PowerReady        bool   `json:"power_ready"`
}

func (p PermitSnapshot) Allowed() bool {
	return p.RouteID != "" && p.MainLocked && p.FlankLocked && p.OverlapLocked && p.CrossingProtected && p.PowerReady
}

type PermitService struct {
	mu      sync.RWMutex
	permits map[string]PermitSnapshot
}

func NewPermitService() *PermitService {
	return &PermitService{permits: map[string]PermitSnapshot{}}
}

func (s *PermitService) Publish(signalID string, snapshot PermitSnapshot) error {
	if signalID == "" || snapshot.RouteID == "" {
		return errors.New("signal and route are required")
	}
	s.mu.Lock()
	s.permits[signalID] = snapshot
	s.mu.Unlock()
	return nil
}

func (s *PermitService) Check(signalID, routeID string) error {
	s.mu.RLock()
	snapshot, ok := s.permits[signalID]
	s.mu.RUnlock()
	if !ok || snapshot.RouteID != routeID {
		return errors.New("signal permit is missing")
	}
	if !snapshot.Allowed() {
		return errors.New("signal permit is not fail-safe complete")
	}
	return nil
}

func (s *PermitService) Revoke(signalID string) {
	s.mu.Lock()
	delete(s.permits, signalID)
	s.mu.Unlock()
}

func (s *PermitService) Snapshot(signalID string) (PermitSnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.permits[signalID]
	return value, ok
}

func (s *PermitService) FailSafe(signalID string, state model.SignalState) bool {
	if state.Commanded == model.AspectProceed && !state.AspectProved() {
		s.Revoke(signalID)
		return false
	}
	return true
}
