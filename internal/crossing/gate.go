package crossing

import (
	"errors"
	"sync"
	"time"
)

type GatePosition string

const (
	GateUp         GatePosition = "up"
	GateMovingDown GatePosition = "moving-down"
	GateDown       GatePosition = "down"
	GateMovingUp   GatePosition = "moving-up"
)

type GateState struct {
	ID            string       `json:"id"`
	Position      GatePosition `json:"position"`
	ProofSession  string       `json:"proof_session,omitempty"`
	ProofRevision uint64       `json:"proof_revision"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

type Gate struct {
	mu    sync.RWMutex
	state GateState
}

func NewGate(id string) *Gate {
	return &Gate{state: GateState{ID: id, Position: GateUp, UpdatedAt: time.Now().UTC()}}
}

func (g *Gate) Move(sessionID string, position GatePosition, at time.Time) (GateState, error) {
	if sessionID == "" {
		return GateState{}, errors.New("crossing session is required")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.state.Position = position
	g.state.UpdatedAt = at
	if position == GateDown {
		g.state.ProofSession = sessionID
		g.state.ProofRevision++
	}
	return g.state, nil
}

func (g *Gate) DownProof(sessionID string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.state.ProofSession == sessionID
}

func (g *Gate) Snapshot() GateState {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.state
}
