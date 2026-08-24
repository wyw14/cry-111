package route

import (
	"errors"

	"github.com/wyw14/cry-111/internal/flank"
	"github.com/wyw14/cry-111/internal/interlock"
	"github.com/wyw14/cry-111/internal/model"
	"github.com/wyw14/cry-111/internal/overlap"
)

type TransactionResult struct {
	Reservation interlock.Reservation `json:"reservation"`
	Decision    interlock.Decision    `json:"decision"`
	Flank       []flank.Protection    `json:"flank"`
	Overlap     overlap.Reservation   `json:"overlap"`
}

type Transaction struct {
	engine  *interlock.Engine
	flank   *flank.ProtectionService
	overlap *overlap.ReservationStore
}

func NewTransaction(engine *interlock.Engine, flankService *flank.ProtectionService, overlapStore *overlap.ReservationStore) *Transaction {
	return &Transaction{engine: engine, flank: flankService, overlap: overlapStore}
}

func (t *Transaction) Commit(route model.Route) (TransactionResult, error) {
	mainOnly := route.Clone()
	mainOnly.OverlapSections = nil
	reservation, decision, err := t.engine.LockRoute(mainOnly)
	if err != nil {
		return TransactionResult{Decision: decision}, err
	}
	flankProtection, err := t.flank.Establish(route.ID, route.FlankPoints)
	if err != nil {
		t.engine.UnlockRoute(route.ID)
		return TransactionResult{Reservation: reservation, Decision: decision}, err
	}
	overlapReservation, err := t.overlap.Reserve(route.ID, route.OverlapSections)
	if err != nil {
		t.flank.Release(route.ID)
		t.engine.UnlockRoute(route.ID)
		return TransactionResult{Reservation: reservation, Decision: decision, Flank: flankProtection}, err
	}
	if len(reservation.Resources) == 0 {
		t.overlap.Release(route.ID)
		t.flank.Release(route.ID)
		t.engine.UnlockRoute(route.ID)
		return TransactionResult{}, errors.New("empty route reservation")
	}
	return TransactionResult{Reservation: reservation, Decision: decision, Flank: flankProtection, Overlap: overlapReservation}, nil
}

func (t *Transaction) Rollback(routeID string) {
	t.overlap.Release(routeID)
	t.flank.Release(routeID)
	t.engine.UnlockRoute(routeID)
}
