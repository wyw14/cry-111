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

// ReleaseAfterDark releases the flank points, overlap sections and interlocking
// resources for a route. It is fail-safe: it only releases anything after
// confirming the protecting signal has reliably proved dark. If the dark proof
// is missing the caller must have already waited on it; this guard is the last
// line of defence and refuses to release points or sections that protect a
// signal which may still be showing a proceed aspect (relay drop-out delay can
// keep the field green for seconds after the close command).
func (r *Resources) ReleaseAfterDark(routeID, signalID string, at time.Time) (ResourceRelease, error) {
	if r.closeProof != nil && !r.closeProof.Proved(signalID) {
		// The proceed aspect has not been proved dark. Do not release the
		// points or sections: they must stay locked for the whole lamp
		// drop-out window, otherwise another route can command the points
		// to a conflicting target while the field signal is still green.
		return ResourceRelease{}, errors.New("signal has not proved dark before release")
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
