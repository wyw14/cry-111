package route

import (
	"errors"
	"sync"

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
	// commitMu serializes the whole multi-resource reservation so that two
	// concurrent route requests cannot interleave their main/flank/overlap
	// sub-reservations. Without it, two requests can both pass the clear-but-
	// unreserved checks, both reserve their main sections, and only collide
	// later at the overlap step — by which point both pages already show
	// "route locked" and the loser keeps a partial lock.
	commitMu sync.Mutex
}

func NewTransaction(engine *interlock.Engine, flankService *flank.ProtectionService, overlapStore *overlap.ReservationStore) *Transaction {
	return &Transaction{engine: engine, flank: flankService, overlap: overlapStore}
}

// Commit reserves the main route, its flank protection and its overlap sections
// as a single atomic decision. The full route (overlap included) is handed to
// the interlocking engine so that main sections, overlap sections, flank points
// and the signal are all checked for conflict under one resource-store lock in
// one pass; a failure at any later step rolls back every resource reserved so
// far, so a failed request never retains a partial lock.
func (t *Transaction) Commit(route model.Route) (TransactionResult, error) {
	t.commitMu.Lock()
	defer t.commitMu.Unlock()

	// Reserve the complete safety plan (main + overlap + flank + signal) under
	// one atomic conflict check. Passing the full route — rather than a clone
	// with OverlapSections cleared — means an overlap section a competing route
	// is entering is detected here, before any partial lock is observable.
	reservation, decision, err := t.engine.LockRoute(route)
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

// Rollback releases every resource a route holds. It takes the commit lock so a
// rollback cannot race with a concurrent commit of a different route that
// happens to touch the same shared resource stores.
func (t *Transaction) Rollback(routeID string) {
	t.commitMu.Lock()
	defer t.commitMu.Unlock()
	t.overlap.Release(routeID)
	t.flank.Release(routeID)
	t.engine.UnlockRoute(routeID)
}
