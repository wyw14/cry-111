package power

import (
	"errors"
	"sort"
	"sync"
	"time"
)

type DomainState string

const (
	DomainOffline     DomainState = "offline"
	DomainSelfTesting DomainState = "self-testing"
	DomainReady       DomainState = "ready"
	DomainFailed      DomainState = "failed"
)

type Domain struct {
	ID        string      `json:"id"`
	State     DomainState `json:"state"`
	Revision  uint64      `json:"revision"`
	UpdatedAt time.Time   `json:"updated_at"`
	Detail    string      `json:"detail,omitempty"`
}

type DomainSet struct {
	mu       sync.RWMutex
	domains  map[string]Domain
	required []string
}

func NewDomainSet(required []string) *DomainSet {
	domains := make(map[string]Domain, len(required))
	now := time.Now().UTC()
	for _, id := range required {
		domains[id] = Domain{ID: id, State: DomainOffline, UpdatedAt: now}
	}
	copyRequired := append([]string(nil), required...)
	sort.Strings(copyRequired)
	return &DomainSet{domains: domains, required: copyRequired}
}

func (s *DomainSet) Transition(id string, state DomainState, detail string, at time.Time) (Domain, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	domain, ok := s.domains[id]
	if !ok {
		return Domain{}, errors.New("unknown power domain")
	}
	if !validTransition(domain.State, state) {
		return Domain{}, errors.New("invalid power domain transition")
	}
	domain.State = state
	domain.Detail = detail
	domain.Revision++
	domain.UpdatedAt = at
	s.domains[id] = domain
	return domain, nil
}

func (s *DomainSet) MarkReady(id string, at time.Time) (Domain, error) {
	return s.Transition(id, DomainReady, "self-test complete", at)
}

func (s *DomainSet) AllReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, id := range s.required {
		if s.domains[id].State != DomainReady {
			return false
		}
	}
	return true
}

func (s *DomainSet) Ready(ids []string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, id := range ids {
		if s.domains[id].State != DomainReady {
			return false
		}
	}
	return true
}

func (s *DomainSet) List() []Domain {
	s.mu.RLock()
	items := make([]Domain, 0, len(s.domains))
	for _, domain := range s.domains {
		items = append(items, domain)
	}
	s.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func validTransition(from, to DomainState) bool {
	if from == to {
		return true
	}
	switch from {
	case DomainOffline:
		return to == DomainSelfTesting || to == DomainFailed
	case DomainSelfTesting:
		return to == DomainReady || to == DomainFailed || to == DomainOffline
	case DomainReady:
		return to == DomainOffline || to == DomainFailed || to == DomainSelfTesting
	case DomainFailed:
		return to == DomainOffline || to == DomainSelfTesting
	default:
		return false
	}
}
