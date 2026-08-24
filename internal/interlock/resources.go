package interlock

import (
	"errors"
	"sort"
	"sync"
)

type Reservation struct {
	Owner     string   `json:"owner"`
	Resources []string `json:"resources"`
	Epoch     uint64   `json:"epoch"`
}

type Resources struct {
	mu      sync.Mutex
	owners  map[string]string
	byOwner map[string][]string
	epoch   uint64
}

func NewResources() *Resources {
	return &Resources{owners: map[string]string{}, byOwner: map[string][]string{}}
}

func (r *Resources) Reserve(owner string, resources []string) (Reservation, error) {
	requested := uniqueResources(resources)
	if owner == "" || len(requested) == 0 {
		return Reservation{}, errors.New("reservation owner and resources are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, resource := range requested {
		if current := r.owners[resource]; current != "" && current != owner {
			return Reservation{}, errors.New("interlocking resource conflict")
		}
	}
	r.epoch++
	for _, resource := range requested {
		r.owners[resource] = owner
	}
	r.byOwner[owner] = requested
	return Reservation{Owner: owner, Resources: append([]string(nil), requested...), Epoch: r.epoch}, nil
}

func (r *Resources) Release(owner string) []string {
	r.mu.Lock()
	resources := append([]string(nil), r.byOwner[owner]...)
	delete(r.byOwner, owner)
	for _, resource := range resources {
		if r.owners[resource] == owner {
			delete(r.owners, resource)
		}
	}
	if len(resources) > 0 {
		r.epoch++
	}
	r.mu.Unlock()
	return resources
}

func (r *Resources) Owner(resource string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.owners[resource]
}

func (r *Resources) Snapshot(owner string) Reservation {
	r.mu.Lock()
	defer r.mu.Unlock()
	return Reservation{Owner: owner, Resources: append([]string(nil), r.byOwner[owner]...), Epoch: r.epoch}
}

func (r *Resources) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.owners)
}

func uniqueResources(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		if value != "" {
			set[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
