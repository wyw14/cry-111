package track

import (
	"errors"
	"sync"
	"time"

	"github.com/wyw14/cry-111/internal/model"
)

type BaselineApplication struct {
	SectionID string    `json:"section_id"`
	ResetID   string    `json:"reset_id"`
	Occupied  bool      `json:"occupied"`
	AppliedAt time.Time `json:"applied_at"`
}

type Section struct {
	store   *Store
	mu      sync.RWMutex
	history []BaselineApplication
}

func NewSection(store *Store) *Section {
	return &Section{store: store}
}

func (s *Section) ApplyBaseline(sectionID, resetID string, occupied bool, at time.Time) (model.TrackState, error) {
	if sectionID == "" || resetID == "" {
		return model.TrackState{}, errors.New("section and reset identities are required")
	}
	state, err := s.store.SetOccupied(sectionID, occupied, "axle-reset", at)
	if err != nil {
		return model.TrackState{}, err
	}
	application := BaselineApplication{SectionID: sectionID, ResetID: resetID, Occupied: occupied, AppliedAt: at}
	s.mu.Lock()
	s.history = append(s.history, application)
	s.mu.Unlock()
	return state, nil
}

func (s *Section) History() []BaselineApplication {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]BaselineApplication(nil), s.history...)
}
