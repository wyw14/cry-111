package release

import (
	"errors"
	"sync"
	"time"

	"github.com/wyw14/cry-111/internal/flank"
	"github.com/wyw14/cry-111/internal/interlock"
	"github.com/wyw14/cry-111/internal/overlap"
	"github.com/wyw14/cry-111/internal/signal"
)

type ResourceRelease struct {
	RouteID   string    `json:"route_id"`
	SignalID  string    `json:"signal_id"`
	Resources []string  `json:"resources"`
	Points    []string  `json:"points"`
	Overlap   []string  `json:"overlap"`
	At        time.Time `json:"at"`
}

type Resources struct {
	interlocking *interlock.Engine
	flank        *flank.ProtectionService
	overlap      *overlap.ReservationStore
	closeProof   *signal.CloseProof
	mu           sync.RWMutex
	history      []ResourceRelease
}

func NewResources(engine *interlock.Engine, flankService *flank.ProtectionService, overlapStore *overlap.ReservationStore, closeProof *signal.CloseProof) *Resources {
	return &Resources{interlocking: engine, flank: flankService, overlap: overlapStore, closeProof: closeProof}
}

func (r *Resources) ReleaseAfterDark(routeID, signalID string, at time.Time) (ResourceRelease, error) {
	if !r.closeProof.Proved(signalID) {
		return ResourceRelease{}, errors.New("signal dark proof is required before route release")
	}
	points := r.flank.Release(routeID)
	overlapSections := r.overlap.Release(routeID)
	decision := r.interlocking.UnlockRoute(routeID)
	if !decision.Allowed {
		return ResourceRelease{}, errors.New(decision.Reason)
	}
	release := ResourceRelease{RouteID: routeID, SignalID: signalID, Resources: []string{"route:" + routeID}, Points: points, Overlap: overlapSections, At: at}
	r.mu.Lock()
	r.history = append(r.history, release)
	r.mu.Unlock()
	return release, nil
}

func (r *Resources) History() []ResourceRelease {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]ResourceRelease, len(r.history))
	copy(items, r.history)
	return items
}
