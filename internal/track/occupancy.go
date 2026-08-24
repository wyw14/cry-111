package track

import (
	"errors"
	"sync"
	"time"

	"github.com/wyw14/cry-111/internal/model"
)

type OccupancyService struct {
	store     *Store
	mu        sync.Mutex
	incidents []model.Incident
}

func NewOccupancyService(store *Store) *OccupancyService {
	return &OccupancyService{store: store}
}

func (s *OccupancyService) Apply(id string, occupied bool, direction string, at time.Time) (model.TrackState, error) {
	if at.IsZero() {
		return model.TrackState{}, errors.New("occupancy timestamp is required")
	}
	state, err := s.store.SetOccupied(id, occupied, direction, at.UTC())
	if err != nil {
		return model.TrackState{}, err
	}
	if occupied && state.ReservedBy == "" {
		s.mu.Lock()
		s.incidents = append(s.incidents, model.NewIncident("critical", "track", id, "unexpected occupancy on unreserved section"))
		s.mu.Unlock()
	}
	return state, nil
}

func (s *OccupancyService) Incidents() []model.Incident {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]model.Incident, len(s.incidents))
	copy(items, s.incidents)
	return items
}

func (s *OccupancyService) Acknowledge(id string, at time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index, incident := range s.incidents {
		if incident.ID == id && incident.Active {
			s.incidents[index] = incident.Acknowledge(at)
			return true
		}
	}
	return false
}
