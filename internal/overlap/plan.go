package overlap

import (
	"errors"
	"sort"
	"sync"
)

type Plan struct {
	RouteName string   `json:"route_name"`
	Sections  []string `json:"sections"`
	Revision  uint64   `json:"revision"`
}

type Planner struct {
	mu    sync.RWMutex
	plans map[string]Plan
}

func NewPlanner() *Planner {
	return &Planner{plans: map[string]Plan{}}
}

func (p *Planner) Configure(routeName string, sections []string, revision uint64) Plan {
	copySections := append([]string(nil), sections...)
	sort.Strings(copySections)
	plan := Plan{RouteName: routeName, Sections: copySections, Revision: revision}
	p.mu.Lock()
	p.plans[routeName] = plan
	p.mu.Unlock()
	return plan
}

func (p *Planner) Resolve(routeName string, topologyRevision uint64) (Plan, error) {
	p.mu.RLock()
	plan, ok := p.plans[routeName]
	p.mu.RUnlock()
	if !ok {
		return Plan{}, errors.New("overlap plan is missing")
	}
	if plan.Revision != topologyRevision {
		return Plan{}, errors.New("overlap plan uses a stale topology revision")
	}
	plan.Sections = append([]string(nil), plan.Sections...)
	return plan, nil
}

func (p *Planner) List() []Plan {
	p.mu.RLock()
	items := make([]Plan, 0, len(p.plans))
	for _, plan := range p.plans {
		plan.Sections = append([]string(nil), plan.Sections...)
		items = append(items, plan)
	}
	p.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].RouteName < items[j].RouteName })
	return items
}
