package overlap

import (
	"sort"
	"sync"
)

type Reservation struct {
	RouteID  string   `json:"route_id"`
	Sections []string `json:"sections"`
}

type ReservationStore struct {
	mu      sync.Mutex
	owners  map[string]string
	byRoute map[string][]string
}

func NewReservationStore() *ReservationStore {
	return &ReservationStore{owners: map[string]string{}, byRoute: map[string][]string{}}
}

func (s *ReservationStore) Reserve(routeID string, sections []string) (Reservation, error) {
	copySections := append([]string(nil), sections...)
	sort.Strings(copySections)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, section := range copySections {
		s.owners[section] = routeID
	}
	s.byRoute[routeID] = copySections
	return Reservation{RouteID: routeID, Sections: append([]string(nil), copySections...)}, nil
}

func (s *ReservationStore) Release(routeID string) []string {
	s.mu.Lock()
	sections := append([]string(nil), s.byRoute[routeID]...)
	delete(s.byRoute, routeID)
	for _, section := range sections {
		if s.owners[section] == routeID {
			delete(s.owners, section)
		}
	}
	s.mu.Unlock()
	return sections
}

func (s *ReservationStore) Owner(section string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.owners[section]
}

func (s *ReservationStore) Snapshot(routeID string) Reservation {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Reservation{RouteID: routeID, Sections: append([]string(nil), s.byRoute[routeID]...)}
}
