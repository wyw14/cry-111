package axlecounter

import (
	"errors"
	"sort"
	"sync"
	"time"
)

type Boundary struct {
	ID        string    `json:"id"`
	Count     int64     `json:"count"`
	Baseline  int64     `json:"baseline"`
	Revision  uint64    `json:"revision"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Section struct {
	ID       string `json:"id"`
	Head     string `json:"head"`
	Tail     string `json:"tail"`
	Occupied bool   `json:"occupied"`
	Revision uint64 `json:"revision"`
}

type Store struct {
	mu         sync.RWMutex
	boundaries map[string]Boundary
	sections   map[string]Section
}

func NewStore() *Store {
	return &Store{boundaries: map[string]Boundary{}, sections: map[string]Section{}}
}

func (s *Store) ConfigureSection(id, head, tail string) error {
	if id == "" || head == "" || tail == "" || head == tail {
		return errors.New("section and two distinct boundaries are required")
	}
	s.mu.Lock()
	s.sections[id] = Section{ID: id, Head: head, Tail: tail}
	if _, ok := s.boundaries[head]; !ok {
		s.boundaries[head] = Boundary{ID: head}
	}
	if _, ok := s.boundaries[tail]; !ok {
		s.boundaries[tail] = Boundary{ID: tail}
	}
	s.mu.Unlock()
	return nil
}

func (s *Store) Count(boundaryID string, delta int64, at time.Time) (Boundary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	boundary, ok := s.boundaries[boundaryID]
	if !ok {
		return Boundary{}, errors.New("unknown axle counter boundary")
	}
	boundary.Count += delta
	boundary.Revision++
	boundary.UpdatedAt = at
	s.boundaries[boundaryID] = boundary
	s.recomputeLocked()
	return boundary, nil
}

func (s *Store) snapshotLocked() ([]Boundary, []Section) {
	boundaries := make([]Boundary, 0, len(s.boundaries))
	sections := make([]Section, 0, len(s.sections))
	for _, item := range s.boundaries {
		boundaries = append(boundaries, item)
	}
	for _, item := range s.sections {
		sections = append(sections, item)
	}
	sort.Slice(boundaries, func(i, j int) bool { return boundaries[i].ID < boundaries[j].ID })
	sort.Slice(sections, func(i, j int) bool { return sections[i].ID < sections[j].ID })
	return boundaries, sections
}

func (s *Store) Snapshot() ([]Boundary, []Section) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshotLocked()
}

func (s *Store) recomputeLocked() {
	for id, section := range s.sections {
		head := s.boundaries[section.Head]
		tail := s.boundaries[section.Tail]
		section.Occupied = head.Count-head.Baseline != tail.Count-tail.Baseline
		section.Revision++
		s.sections[id] = section
	}
}
