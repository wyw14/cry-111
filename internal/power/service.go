package power

import (
	"sync"
	"time"

	"github.com/wyw14/cry-111/internal/model"
)

type Service struct {
	domains   *DomainSet
	mu        sync.Mutex
	incidents []model.Incident
}

func NewService(domains *DomainSet) *Service {
	return &Service{domains: domains}
}

func (s *Service) Lose(id string, at time.Time) (Domain, error) {
	domain, err := s.domains.Transition(id, DomainOffline, "supply lost", at)
	if err == nil {
		s.mu.Lock()
		s.incidents = append(s.incidents, model.NewIncident("critical", "power", id, "interlocking power domain unavailable"))
		s.mu.Unlock()
	}
	return domain, err
}

func (s *Service) BeginRecovery(id string, at time.Time) (Domain, error) {
	return s.domains.Transition(id, DomainSelfTesting, "self-test running", at)
}

func (s *Service) CompleteRecovery(id string, at time.Time) (Domain, error) {
	return s.domains.MarkReady(id, at)
}

func (s *Service) StartAll(at time.Time) error {
	for _, domain := range s.domains.List() {
		if domain.State == DomainOffline || domain.State == DomainFailed {
			if _, err := s.BeginRecovery(domain.ID, at); err != nil {
				return err
			}
		}
		if _, err := s.CompleteRecovery(domain.ID, at.Add(time.Millisecond)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Incidents() []model.Incident {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]model.Incident(nil), s.incidents...)
}

func (s *Service) Domains() *DomainSet {
	return s.domains
}
