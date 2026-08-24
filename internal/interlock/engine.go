package interlock

import (
	"errors"
	"sync"
	"time"

	"github.com/wyw14/cry-111/internal/model"
)

type Decision struct {
	ID      string    `json:"id"`
	RouteID string    `json:"route_id"`
	Allowed bool      `json:"allowed"`
	Reason  string    `json:"reason"`
	At      time.Time `json:"at"`
}

type Engine struct {
	resources *Resources
	occupancy *OccupancyPermit
	mu        sync.Mutex
	decisions []Decision
}

func NewEngine(resources *Resources, occupancy *OccupancyPermit) *Engine {
	return &Engine{resources: resources, occupancy: occupancy}
}

func (e *Engine) LockRoute(route model.Route) (Reservation, Decision, error) {
	if err := e.occupancy.RequireClear(append(append([]string(nil), route.Sections...), route.OverlapSections...)); err != nil {
		decision := e.record(route.ID, false, err.Error())
		return Reservation{}, decision, err
	}
	reservation, err := e.resources.Reserve(route.ID, route.Resources())
	if err != nil {
		decision := e.record(route.ID, false, err.Error())
		return Reservation{}, decision, err
	}
	decision := e.record(route.ID, true, "complete safety plan reserved")
	return reservation, decision, nil
}

func (e *Engine) UnlockRoute(routeID string) Decision {
	released := e.resources.Release(routeID)
	if len(released) == 0 {
		return e.record(routeID, false, "route had no resources")
	}
	return e.record(routeID, true, "route resources released")
}

func (e *Engine) Reservation(routeID string) (Reservation, error) {
	value := e.resources.Snapshot(routeID)
	if len(value.Resources) == 0 {
		return Reservation{}, errors.New("route reservation not found")
	}
	return value, nil
}

func (e *Engine) Decisions() []Decision {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]Decision(nil), e.decisions...)
}

func (e *Engine) record(routeID string, allowed bool, reason string) Decision {
	decision := Decision{ID: model.NewIdentity().String(), RouteID: routeID, Allowed: allowed, Reason: reason, At: time.Now().UTC()}
	e.mu.Lock()
	e.decisions = append(e.decisions, decision)
	e.mu.Unlock()
	return decision
}
