package overlap

import (
	"errors"
	"sort"
	"sync"
)

type Reservation struct {
	RouteID  string   `json:"route_id"`
	Sections []string `json:"sections"`
}

// ErrOverlapConflict is returned when an overlap section is already owned by a
// different route. It lets a transaction reject the whole reservation atomically
// instead of silently overwriting the previous owner.
var ErrOverlapConflict = errors.New("overlap section conflict")

type ReservationStore struct {
	mu      sync.Mutex
	owners  map[string]string
	byRoute map[string][]string
}

func NewReservationStore() *ReservationStore {
	return &ReservationStore{owners: map[string]string{}, byRoute: map[string][]string{}}
}

// Reserve atomically claims the overlap sections for routeID. A section already
// owned by another route fails the whole reservation with ErrOverlapConflict and
// leaves the store untouched, so a failed request never retains a partial
// overlap lock that a competing request could observe as "route locked".
func (s *ReservationStore) Reserve(routeID string, sections []string) (Reservation, error) {
	if routeID == "" {
		return Reservation{}, errors.New("overlap reservation requires a route id")
	}
	copySections := uniqueSections(sections)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, section := range copySections {
		if current := s.owners[section]; current != "" && current != routeID {
			return Reservation{}, ErrOverlapConflict
		}
	}
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

func uniqueSections(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		if value != "" {
			set[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
