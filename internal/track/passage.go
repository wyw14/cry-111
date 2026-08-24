package track

import (
	"sync"
	"time"
)

type PassagePhase string

const (
	PassageWaiting  PassagePhase = "waiting"
	PassageEntered  PassagePhase = "entered"
	PassageAdvanced PassagePhase = "advanced"
	PassageClearing PassagePhase = "clearing"
	PassageComplete PassagePhase = "complete"
)

type Passage struct {
	RouteID       string       `json:"route_id"`
	Sections      []string     `json:"sections"`
	Direction     string       `json:"direction"`
	Phase         PassagePhase `json:"phase"`
	CurrentIndex  int          `json:"current_index"`
	ClearSince    time.Time    `json:"clear_since"`
	LastEvent     time.Time    `json:"last_event"`
	SequenceValid bool         `json:"sequence_valid"`
}

type PassageTracker struct {
	mu       sync.Mutex
	passages map[string]Passage
	minClear time.Duration
}

func NewPassageTracker(minClear time.Duration) *PassageTracker {
	return &PassageTracker{passages: map[string]Passage{}, minClear: minClear}
}

func (t *PassageTracker) Begin(routeID string, sections []string, direction string, at time.Time) Passage {
	t.mu.Lock()
	defer t.mu.Unlock()
	passage := Passage{RouteID: routeID, Sections: append([]string(nil), sections...), Direction: direction, Phase: PassageWaiting, CurrentIndex: -1, LastEvent: at, SequenceValid: true}
	t.passages[routeID] = passage
	return passage
}

func (t *PassageTracker) Occupy(routeID, sectionID string, at time.Time) (Passage, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	passage, ok := t.passages[routeID]
	if !ok {
		return Passage{}, false
	}
	expected := passage.CurrentIndex + 1
	if expected >= len(passage.Sections) || passage.Sections[expected] != sectionID {
		passage.SequenceValid = false
		passage.LastEvent = at
		t.passages[routeID] = passage
		return passage, false
	}
	passage.CurrentIndex = expected
	passage.ClearSince = time.Time{}
	passage.LastEvent = at
	if expected == 0 {
		passage.Phase = PassageEntered
	} else {
		passage.Phase = PassageAdvanced
	}
	t.passages[routeID] = passage
	return passage, true
}

func (t *PassageTracker) ObserveClear(routeID, sectionID string, at time.Time) (Passage, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	passage, ok := t.passages[routeID]
	if !ok || passage.CurrentIndex < 0 || passage.Sections[passage.CurrentIndex] != sectionID {
		return Passage{}, false
	}
	if passage.ClearSince.IsZero() {
		passage.ClearSince = at
		passage.Phase = PassageClearing
		passage.LastEvent = at
		t.passages[routeID] = passage
		return passage, false
	}
	stable := at.Sub(passage.ClearSince) >= t.minClear
	if stable && passage.SequenceValid && passage.CurrentIndex == len(passage.Sections)-1 {
		passage.Phase = PassageComplete
	}
	passage.LastEvent = at
	t.passages[routeID] = passage
	return passage, stable && passage.SequenceValid
}

func (t *PassageTracker) Snapshot(routeID string) (Passage, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	passage, ok := t.passages[routeID]
	passage.Sections = append([]string(nil), passage.Sections...)
	return passage, ok
}

func (t *PassageTracker) Remove(routeID string) {
	t.mu.Lock()
	delete(t.passages, routeID)
	t.mu.Unlock()
}
