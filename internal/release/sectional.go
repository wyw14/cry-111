package release

import (
	"errors"
	"sync"
	"time"

	"github.com/wyw14/cry-111/internal/track"
)

type SectionRelease struct {
	RouteID      string             `json:"route_id"`
	SectionID    string             `json:"section_id"`
	ReleasedAt   time.Time          `json:"released_at"`
	PassagePhase track.PassagePhase `json:"passage_phase"`
}

type Sectional struct {
	tracker  *track.PassageTracker
	store    *track.Store
	mu       sync.RWMutex
	releases map[string][]SectionRelease
}

func NewSectional(tracker *track.PassageTracker, store *track.Store) *Sectional {
	return &Sectional{tracker: tracker, store: store, releases: map[string][]SectionRelease{}}
}

func (s *Sectional) Begin(routeID string, sections []string, direction string, at time.Time) track.Passage {
	return s.tracker.Begin(routeID, sections, direction, at)
}

func (s *Sectional) Occupy(routeID, sectionID string, at time.Time) error {
	_, valid := s.tracker.Occupy(routeID, sectionID, at)
	if !valid {
		return errors.New("track occupation is out of route sequence")
	}
	return nil
}

func (s *Sectional) ObserveClear(routeID, sectionID string, at time.Time) (SectionRelease, error) {
	passage, releasable := s.tracker.ObserveClear(routeID, sectionID, at)
	if !releasable {
		return SectionRelease{}, errors.New("section does not have a stable complete passage proof")
	}
	state, ok := s.store.Get(sectionID)
	if !ok || state.Occupied {
		return SectionRelease{}, errors.New("track section is not clear")
	}
	release := SectionRelease{RouteID: routeID, SectionID: sectionID, ReleasedAt: at, PassagePhase: passage.Phase}
	s.store.ClearReservation(sectionID, routeID)
	s.mu.Lock()
	s.releases[routeID] = append(s.releases[routeID], release)
	s.mu.Unlock()
	return release, nil
}

func (s *Sectional) Releases(routeID string) []SectionRelease {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]SectionRelease(nil), s.releases[routeID]...)
}

func (s *Sectional) Complete(routeID string) {
	s.tracker.Remove(routeID)
	s.mu.Lock()
	delete(s.releases, routeID)
	s.mu.Unlock()
}
