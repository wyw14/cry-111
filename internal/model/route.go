package model

import "time"

type RoutePhase string

const (
	RouteRequested RoutePhase = "requested"
	RouteProving   RoutePhase = "proving"
	RouteLocked    RoutePhase = "locked"
	RouteSignalled RoutePhase = "signalled"
	RouteOccupied  RoutePhase = "occupied"
	RouteReleasing RoutePhase = "releasing"
	RouteNormal    RoutePhase = "normal"
	RouteCancelled RoutePhase = "cancelled"
)

type Route struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Kind             string     `json:"kind"`
	Phase            RoutePhase `json:"phase"`
	Sections         []string   `json:"sections"`
	Points           []string   `json:"points"`
	FlankPoints      []string   `json:"flank_points"`
	OverlapSections  []string   `json:"overlap_sections"`
	SignalID         string     `json:"signal_id"`
	TopologyRevision uint64     `json:"topology_revision"`
	UpdatedAt        time.Time  `json:"updated_at"`
	Incident         string     `json:"incident,omitempty"`
}

func (r Route) Clone() Route {
	r.Sections = append([]string(nil), r.Sections...)
	r.Points = append([]string(nil), r.Points...)
	r.FlankPoints = append([]string(nil), r.FlankPoints...)
	r.OverlapSections = append([]string(nil), r.OverlapSections...)
	return r
}

func (r Route) Resources() []string {
	items := make([]string, 0, len(r.Sections)+len(r.Points)+len(r.FlankPoints)+len(r.OverlapSections)+1)
	for _, section := range r.Sections {
		items = append(items, "track:"+section)
	}
	for _, point := range r.Points {
		items = append(items, "point:"+point)
	}
	for _, point := range r.FlankPoints {
		items = append(items, "point:"+point)
	}
	for _, section := range r.OverlapSections {
		items = append(items, "track:"+section)
	}
	if r.SignalID != "" {
		items = append(items, "signal:"+r.SignalID)
	}
	return items
}

func (r Route) Active() bool {
	return r.Phase != RouteNormal && r.Phase != RouteCancelled
}
