package interlock

import (
	"errors"
	"sync"

	"github.com/wyw14/cry-111/internal/model"
)

type OccupancyPermit struct {
	mu     sync.RWMutex
	states map[string]model.TrackState
}

func NewOccupancyPermit() *OccupancyPermit {
	return &OccupancyPermit{states: map[string]model.TrackState{}}
}

func (p *OccupancyPermit) Update(state model.TrackState) {
	p.mu.Lock()
	p.states[state.ID] = state
	p.mu.Unlock()
}

func (p *OccupancyPermit) RequireClear(ids []string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, id := range ids {
		state, ok := p.states[id]
		if !ok {
			return errors.New("track occupancy proof is unavailable")
		}
		if state.Occupied {
			return errors.New("track section is occupied")
		}
	}
	return nil
}

func (p *OccupancyPermit) Snapshot() []model.TrackState {
	p.mu.RLock()
	items := make([]model.TrackState, 0, len(p.states))
	for _, state := range p.states {
		items = append(items, state)
	}
	p.mu.RUnlock()
	return items
}
