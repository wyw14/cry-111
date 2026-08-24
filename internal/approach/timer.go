package approach

import (
	"errors"
	"sync"
	"time"
)

type TimeSource interface {
	WallNow() time.Time
	MonotonicElapsed() time.Duration
}

type TimerState struct {
	RouteID         string        `json:"route_id"`
	StartedWall     time.Time     `json:"started_wall"`
	StartedElapsed  time.Duration `json:"started_elapsed"`
	Duration        time.Duration `json:"duration"`
	DisplayDeadline time.Time     `json:"display_deadline"`
}

type Timer struct {
	mu     sync.RWMutex
	clock  TimeSource
	states map[string]TimerState
}

func NewTimer(clock TimeSource) *Timer {
	return &Timer{clock: clock, states: map[string]TimerState{}}
}

func (t *Timer) Start(routeID string, duration time.Duration) (TimerState, error) {
	if routeID == "" || duration <= 0 {
		return TimerState{}, errors.New("route and positive duration are required")
	}
	wall := t.clock.WallNow()
	state := TimerState{RouteID: routeID, StartedWall: wall, StartedElapsed: t.clock.MonotonicElapsed(), Duration: duration, DisplayDeadline: wall.Add(duration)}
	t.mu.Lock()
	t.states[routeID] = state
	t.mu.Unlock()
	return state, nil
}

func (t *Timer) Remaining(routeID string) (time.Duration, error) {
	t.mu.RLock()
	state, ok := t.states[routeID]
	t.mu.RUnlock()
	if !ok {
		return 0, errors.New("approach timer not found")
	}
	elapsed := t.clock.MonotonicElapsed() - state.StartedElapsed
	remaining := state.Duration - elapsed
	if remaining < 0 {
		return 0, nil
	}
	return remaining, nil
}

func (t *Timer) Expired(routeID string) bool {
	remaining, err := t.Remaining(routeID)
	return err == nil && remaining == 0
}

func (t *Timer) Cancel(routeID string) {
	t.mu.Lock()
	delete(t.states, routeID)
	t.mu.Unlock()
}

func (t *Timer) Snapshot(routeID string) (TimerState, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	state, ok := t.states[routeID]
	return state, ok
}
