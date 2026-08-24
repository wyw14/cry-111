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
	// SafetyDeadline is the monotonic-elapsed instant at which the approach
	// lock has run for its full duration. It is measured against the clock's
	// monotonic reference rather than the correctable wall clock, so a
	// station-clock correction can never shorten the safety delay —
	// corrections may only adjust the display/audit times above.
	SafetyDeadline time.Duration `json:"safety_deadline"`
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
	elapsed := t.clock.MonotonicElapsed()
	state := TimerState{
		RouteID:         routeID,
		StartedWall:     wall,
		StartedElapsed:  elapsed,
		Duration:        duration,
		DisplayDeadline: wall.Add(duration),
		// SafetyDeadline is expressed against the monotonic reference so that a
		// station-clock correction cannot shorten the approach-lock delay.
		SafetyDeadline: elapsed + duration,
	}
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
	// Safety remaining is measured against the monotonic reference, which a
	// station-clock correction cannot move; the wall-clock deadline above is
	// display/audit only.
	remaining := state.SafetyDeadline - t.clock.MonotonicElapsed()
	if remaining < 0 {
		return 0, nil
	}
	return remaining, nil
}

func (t *Timer) Expired(routeID string) bool {
	remaining, err := t.Remaining(routeID)
	return err == nil && remaining == 0
}

// DisplayRemaining returns the wall-clock remaining duration. It is meant for
// operator display and audit only; because it is derived from the correctable
// wall clock, a station-clock correction shifts it. It must never gate the
// approach lock — use Remaining/Expired for the safety decision.
func (t *Timer) DisplayRemaining(routeID string) (time.Duration, error) {
	t.mu.RLock()
	state, ok := t.states[routeID]
	t.mu.RUnlock()
	if !ok {
		return 0, errors.New("approach timer not found")
	}
	remaining := state.DisplayDeadline.Sub(t.clock.WallNow())
	if remaining < 0 {
		return 0, nil
	}
	return remaining, nil
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
