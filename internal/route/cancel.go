package route

import (
	"errors"
	"time"

	"github.com/wyw14/cry-111/internal/approach"
	"github.com/wyw14/cry-111/internal/model"
)

type Cancellation struct {
	RouteID     string    `json:"route_id"`
	RequestedAt time.Time `json:"requested_at"`
	Allowed     bool      `json:"allowed"`
	Reason      string    `json:"reason"`
}

type CancellationService struct {
	routes      *Store
	locks       *approach.LockingService
	timers      *approach.Timer
	transaction *Transaction
}

func NewCancellationService(routes *Store, locks *approach.LockingService, timers *approach.Timer, transaction *Transaction) *CancellationService {
	return &CancellationService{routes: routes, locks: locks, timers: timers, transaction: transaction}
}

func (s *CancellationService) Request(routeID string, at time.Time) (Cancellation, error) {
	route, ok := s.routes.Get(routeID)
	if !ok {
		return Cancellation{}, errors.New("route not found")
	}
	if s.locks.Active(routeID) && !s.timers.Expired(routeID) {
		remaining, _ := s.timers.Remaining(routeID)
		return Cancellation{RouteID: routeID, RequestedAt: at, Allowed: false, Reason: "approach locking active for " + remaining.String()}, errors.New("approach locking prevents cancellation")
	}
	if route.Phase == model.RouteSignalled || route.Phase == model.RouteOccupied {
		return Cancellation{RouteID: routeID, RequestedAt: at, Allowed: false, Reason: "emergency cancellation is required"}, errors.New("route has an open or occupied signal")
	}
	s.transaction.Rollback(routeID)
	_, err := s.routes.Transition(routeID, model.RouteCancelled, "operator cancellation")
	if err != nil {
		return Cancellation{}, err
	}
	s.locks.Release(routeID)
	s.timers.Cancel(routeID)
	return Cancellation{RouteID: routeID, RequestedAt: at, Allowed: true, Reason: "route safely cancelled"}, nil
}

func (s *CancellationService) EngageApproach(routeID, sectionID string, duration time.Duration, at time.Time) error {
	if _, err := s.locks.Engage(routeID, sectionID, at, duration); err != nil {
		return err
	}
	_, err := s.timers.Start(routeID, duration)
	return err
}
