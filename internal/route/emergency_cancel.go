package route

import (
	"context"
	"errors"
	"time"

	"github.com/wyw14/cry-111/internal/model"
	"github.com/wyw14/cry-111/internal/release"
	"github.com/wyw14/cry-111/internal/signal"
)

type EmergencyCancellation struct {
	RouteID    string    `json:"route_id"`
	SignalID   string    `json:"signal_id"`
	DarkProved bool      `json:"dark_proved"`
	Released   bool      `json:"released"`
	At         time.Time `json:"at"`
}

type EmergencyCanceller struct {
	routes     *Store
	closeProof *signal.CloseProof
	release    *release.Resources
}

func NewEmergencyCanceller(routes *Store, closeProof *signal.CloseProof, releaseResources *release.Resources) *EmergencyCanceller {
	return &EmergencyCanceller{routes: routes, closeProof: closeProof, release: releaseResources}
}

func (c *EmergencyCanceller) Request(ctx context.Context, routeID string, at time.Time) (EmergencyCancellation, error) {
	route, ok := c.routes.Get(routeID)
	if !ok {
		return EmergencyCancellation{}, errors.New("route not found")
	}
	if route.Phase != model.RouteSignalled && route.Phase != model.RouteOccupied && route.Phase != model.RouteLocked {
		return EmergencyCancellation{}, errors.New("route is not eligible for emergency cancellation")
	}
	if err := c.closeProof.Request(route.SignalID, route.ID, at); err != nil {
		return EmergencyCancellation{}, err
	}
	if _, err := c.routes.Transition(route.ID, model.RouteReleasing, "emergency cancellation"); err != nil {
		return EmergencyCancellation{}, err
	}
	if _, err := c.release.ReleaseAfterDark(route.ID, route.SignalID, at); err != nil {
		return EmergencyCancellation{RouteID: route.ID, SignalID: route.SignalID, At: at}, err
	}
	if err := c.closeProof.Wait(ctx, route.SignalID); err != nil {
		return EmergencyCancellation{RouteID: route.ID, SignalID: route.SignalID, Released: true, At: at}, err
	}
	if _, err := c.routes.Transition(route.ID, model.RouteCancelled, "emergency cancellation complete"); err != nil {
		return EmergencyCancellation{}, err
	}
	return EmergencyCancellation{RouteID: route.ID, SignalID: route.SignalID, DarkProved: true, Released: true, At: at}, nil
}

func (c *EmergencyCanceller) ConfirmDark(signalID string, at time.Time) error {
	state, err := c.closeProof.ObserveDark(signalID, at)
	if err != nil {
		return err
	}
	if !state.DarkProved {
		return errors.New("signal did not prove dark")
	}
	return nil
}
