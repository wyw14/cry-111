package crossing

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID            string    `json:"id"`
	RouteID       string    `json:"route_id"`
	State         string    `json:"state"`
	StartedAt     time.Time `json:"started_at"`
	ProofRevision uint64    `json:"proof_revision"`
}

type Controller struct {
	gate     *Gate
	mu       sync.RWMutex
	sessions map[string]Session
}

func NewController(gate *Gate) *Controller {
	return &Controller{gate: gate, sessions: map[string]Session{}}
}

func (c *Controller) Close(routeID string, at time.Time) (Session, error) {
	if routeID == "" {
		return Session{}, errors.New("route is required")
	}
	session := Session{ID: uuid.NewString(), RouteID: routeID, State: "closing", StartedAt: at}
	if _, err := c.gate.Move(session.ID, GateMovingDown, at); err != nil {
		return Session{}, err
	}
	c.mu.Lock()
	c.sessions[session.ID] = session
	c.mu.Unlock()
	return session, nil
}

func (c *Controller) ProveDown(sessionID string, at time.Time) (Session, error) {
	c.mu.Lock()
	session, ok := c.sessions[sessionID]
	if !ok {
		c.mu.Unlock()
		return Session{}, errors.New("crossing session not found")
	}
	state, err := c.gate.Move(sessionID, GateDown, at)
	if err != nil {
		c.mu.Unlock()
		return Session{}, err
	}
	session.State = "protected"
	session.ProofRevision = state.ProofRevision
	c.sessions[sessionID] = session
	c.mu.Unlock()
	return session, nil
}

func (c *Controller) Open(sessionID string, at time.Time) (Session, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	session, ok := c.sessions[sessionID]
	if !ok {
		return Session{}, errors.New("crossing session not found")
	}
	if _, err := c.gate.Move(sessionID, GateMovingUp, at); err != nil {
		return Session{}, err
	}
	session.State = "opening"
	session.ProofRevision = 0
	c.sessions[sessionID] = session
	return session, nil
}

func (c *Controller) Reclose(previousSessionID, routeID string, at time.Time) (Session, error) {
	c.mu.RLock()
	_, exists := c.sessions[previousSessionID]
	c.mu.RUnlock()
	if !exists {
		return Session{}, errors.New("previous crossing session not found")
	}
	return c.Close(routeID, at)
}

func (c *Controller) Protected(sessionID string) bool {
	c.mu.RLock()
	session, ok := c.sessions[sessionID]
	c.mu.RUnlock()
	return ok && session.State == "protected" && c.gate.DownProof(sessionID)
}

func (c *Controller) List() []Session {
	c.mu.RLock()
	items := make([]Session, 0, len(c.sessions))
	for _, session := range c.sessions {
		items = append(items, session)
	}
	c.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].StartedAt.Before(items[j].StartedAt) })
	return items
}
