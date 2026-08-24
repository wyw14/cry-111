package route

import (
	"errors"
	"sync"

	"github.com/wyw14/cry-111/internal/model"
	"github.com/wyw14/cry-111/internal/power"
)

type ReproveResult struct {
	RouteID         string           `json:"route_id"`
	Phase           model.RoutePhase `json:"phase"`
	AllDomainsReady bool             `json:"all_domains_ready"`
	ProofValid      bool             `json:"proof_valid"`
}

type Reprover struct {
	domains  *power.DomainSet
	store    *Store
	prover   *Prover
	required []string
	mu       sync.RWMutex
	results  map[string]ReproveResult
}

func NewReprover(domains *power.DomainSet, store *Store, prover *Prover, required []string) *Reprover {
	return &Reprover{domains: domains, store: store, prover: prover, required: append([]string(nil), required...), results: map[string]ReproveResult{}}
}

func (r *Reprover) RestoreProving(route model.Route) error {
	if !r.domains.Ready(r.required) {
		return errors.New("not all interlocking power domains are ready")
	}
	route.Phase = safeRecoveryPhase(route)
	if err := r.store.Put(route); err != nil {
		return err
	}
	_, err := r.prover.Prove(route)
	result := ReproveResult{RouteID: route.ID, Phase: route.Phase, AllDomainsReady: true, ProofValid: err == nil}
	r.mu.Lock()
	r.results[route.ID] = result
	r.mu.Unlock()
	return err
}

func (r *Reprover) Result(routeID string) (ReproveResult, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result, ok := r.results[routeID]
	return result, ok
}

func (r *Reprover) RequiredDomains() []string {
	return append([]string(nil), r.required...)
}
