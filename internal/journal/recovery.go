package journal

import (
	"errors"

	"github.com/wyw14/cry-111/internal/model"
	"github.com/wyw14/cry-111/internal/power"
)

type RouteRestorer interface {
	RestoreProving(model.Route) error
}

type Recovery struct {
	domains  *power.DomainSet
	restorer RouteRestorer
	required []string
}

func NewRecovery(domains *power.DomainSet, restorer RouteRestorer, required []string) *Recovery {
	return &Recovery{domains: domains, restorer: restorer, required: append([]string(nil), required...)}
}

func (r *Recovery) RestoreRoutes(routes []model.Route) error {
	if !r.domains.Ready(r.required) {
		return errors.New("hardware power domains are not ready for route recovery")
	}
	for _, route := range routes {
		route.Phase = model.RouteProving
		if err := r.restorer.RestoreProving(route); err != nil {
			return err
		}
	}
	return nil
}

func (r *Recovery) RequiredDomains() []string {
	return append([]string(nil), r.required...)
}
