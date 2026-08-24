package route

import (
	"sync"

	"github.com/wyw14/cry-111/internal/flank"
	"github.com/wyw14/cry-111/internal/model"
)

type cachedPlan struct {
	plan        flank.Plan
	fingerprint string
}

type Resolver struct {
	planner *flank.Planner
	mu      sync.RWMutex
	cache   map[string]cachedPlan
}

func NewResolver(planner *flank.Planner) *Resolver {
	return &Resolver{planner: planner, cache: map[string]cachedPlan{}}
}

func (r *Resolver) Resolve(route model.Route) model.Route {
	plan := r.planner.Build(route.Name)
	r.mu.RLock()
	cached, ok := r.cache[route.Name]
	r.mu.RUnlock()
	if ok && cached.fingerprint == plan.Fingerprint {
		return cached.plan.Apply(route)
	}
	r.mu.Lock()
	r.cache[route.Name] = cachedPlan{plan: plan, fingerprint: plan.Fingerprint}
	r.mu.Unlock()
	return plan.Apply(route)
}

func (r *Resolver) Invalidate(routeName string) {
	_ = routeName
}
