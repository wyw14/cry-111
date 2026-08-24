package route

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wyw14/cry-111/internal/flank"
	"github.com/wyw14/cry-111/internal/interlock"
	"github.com/wyw14/cry-111/internal/model"
	"github.com/wyw14/cry-111/internal/overlap"
	"github.com/wyw14/cry-111/internal/point"
)

// newTestTransaction wires a Transaction against fresh, shared resource stores so
// that two routes competing for the same overlap section are adjudicated through
// the same interlock resource pool and overlap store.
func newTestTransaction(t *testing.T, trackIDs, pointIDs []string) (*Transaction, *interlock.Resources, *overlap.ReservationStore) {
	t.Helper()
	occupancy := interlock.NewOccupancyPermit()
	now := time.Now().UTC()
	for _, id := range trackIDs {
		occupancy.Update(model.TrackState{ID: id, StableSince: now, LastTransition: now})
	}
	pointStore := point.NewStore(pointIDs)
	proofReader := point.NewProofReader(pointStore)
	pointLocks := point.NewLockManager(pointStore)
	resources := interlock.NewResources()
	engine := interlock.NewEngine(resources, occupancy)
	flankService := flank.NewProtectionService(proofReader, pointLocks)
	overlapStore := overlap.NewReservationStore()
	return NewTransaction(engine, flankService, overlapStore), resources, overlapStore
}

func makeRoute(id, signal string, sections, overlap []string) model.Route {
	return model.Route{
		ID:               id,
		Name:             id,
		Kind:             "train",
		SignalID:         signal,
		Sections:         sections,
		OverlapSections:  overlap,
		TopologyRevision: 1,
	}
}

// TestCommitAtomicOverlapConflict reproduces the reported race: while a train
// route is locking its overlap (front protection section), a shunting request
// arrives claiming the same section. Before the fix both commits could observe
// "route locked" because the overlap store silently overwrote the owner and the
// main-only reservation never saw the overlap section. After the fix exactly one
// request wins and the loser keeps no partial lock.
func TestCommitAtomicOverlapConflict(t *testing.T) {
	t.Parallel()
	const shared = "5DG"
	transaction, resources, overlapStore := newTestTransaction(t,
		[]string{"1DG", "3DG", shared, "7DG"},
		[]string{"P1", "P2"})

	train := makeRoute("train-A", "S1", []string{"1DG", "3DG"}, []string{shared})
	shunt := makeRoute("shunt-B", "S2", []string{shared, "7DG"}, nil)

	var wg sync.WaitGroup
	var success int32
	var failures int32
	start := make(chan struct{})
	run := func(rt model.Route) {
		defer wg.Done()
		<-start
		if _, err := transaction.Commit(rt); err != nil {
			atomic.AddInt32(&failures, 1)
			return
		}
		atomic.AddInt32(&success, 1)
	}
	wg.Add(2)
	go run(train)
	go run(shunt)
	close(start)
	wg.Wait()

	if got := atomic.LoadInt32(&success); got != 1 {
		t.Fatalf("expected exactly one successful reservation, got %d", got)
	}
	if got := atomic.LoadInt32(&failures); got != 1 {
		t.Fatalf("expected exactly one failed reservation, got %d", got)
	}

	// The shared section must be owned by exactly one route across both stores,
	// never orphaned to a loser that "locked" but then collided.
	trackOwner := resources.Owner("track:" + shared)
	bareOwner := overlapStore.Owner(shared)
	if trackOwner == "" {
		t.Fatalf("shared section %s left unreserved in interlock store", shared)
	}
	if bareOwner != "" && bareOwner != trackOwner {
		t.Fatalf("overlap owner %q disagrees with interlock owner %q for %s", bareOwner, trackOwner, shared)
	}
	// Whichever route won the shared section must also still own the rest of its
	// own main sections — the winner never drops a partial lock to "win" the overlap.
	winnerMain := map[string][]string{
		"train-A": {"track:1DG", "track:3DG"},
		"shunt-B": {"track:7DG"},
	}
	for _, section := range winnerMain[trackOwner] {
		if resources.Owner(section) != trackOwner {
			t.Fatalf("winner %s did not retain its main section %s (owner %q)", trackOwner, section, resources.Owner(section))
		}
	}
	// No other route may hold a stray reservation for sections it never won.
	for _, orphan := range []string{"train-A", "shunt-B"} {
		if orphan == trackOwner {
			continue
		}
		if snap := overlapStore.Snapshot(orphan); len(snap.Sections) != 0 {
			t.Fatalf("loser %s retained partial overlap reservation %v", orphan, snap.Sections)
		}
		if resources.Owner("track:"+shared) == orphan {
			t.Fatalf("loser %s retained a partial reservation of the shared section", orphan)
		}
	}
}

// TestCommitRollbackLeavesNoPartialLock ensures a reservation that fails after
// the main lock (here via an overlap conflict) releases every resource it held,
// leaving the interlock and overlap stores clean for a follow-up request.
func TestCommitRollbackLeavesNoPartialLock(t *testing.T) {
	t.Parallel()
	const shared = "9DG"
	transaction, resources, overlapStore := newTestTransaction(t,
		[]string{"1DG", "3DG", shared, "7DG"},
		[]string{"P1", "P2"})

	established := makeRoute("first", "S1", []string{"1DG", "3DG"}, []string{shared})
	if _, err := transaction.Commit(established); err != nil {
		t.Fatalf("first commit failed: %v", err)
	}

	competing := makeRoute("second", "S2", []string{shared, "7DG"}, nil)
	if _, err := transaction.Commit(competing); err == nil {
		t.Fatalf("competing commit unexpectedly succeeded; overlap conflict not detected")
	}
	// The loser must hold nothing in either store.
	if resources.Owner("track:7DG") == "second" {
		t.Fatalf("loser retained a partial main-section lock on 7DG")
	}
	if got := overlapStore.Snapshot("second"); len(got.Sections) != 0 {
		t.Fatalf("loser retained a partial overlap reservation %v", got.Sections)
	}
}

// TestOverlapReserveDetectsConflict guards the overlap store directly: the old
// implementation silently overwrote the owner, so a second reservation would
// report success and both routes would show "locked".
func TestOverlapReserveDetectsConflict(t *testing.T) {
	t.Parallel()
	store := overlap.NewReservationStore()
	if _, err := store.Reserve("R1", []string{"OV1", "OV2"}); err != nil {
		t.Fatalf("first reserve failed: %v", err)
	}
	if _, err := store.Reserve("R2", []string{"OV2"}); !errors.Is(err, overlap.ErrOverlapConflict) {
		t.Fatalf("expected ErrOverlapConflict, got %v", err)
	}
	if owner := store.Owner("OV2"); owner != "R1" {
		t.Fatalf("conflicting reservation overwrote owner to %q", owner)
	}
}

