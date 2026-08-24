package flank

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"sync"

	"github.com/wyw14/cry-111/internal/model"
)

type Topology struct {
	Revision uint64              `json:"revision"`
	Main     map[string][]string `json:"main"`
	Flank    map[string][]string `json:"flank"`
	Overlap  map[string][]string `json:"overlap"`
}

type Plan struct {
	RouteName       string   `json:"route_name"`
	Revision        uint64   `json:"revision"`
	MainSections    []string `json:"main_sections"`
	FlankPoints     []string `json:"flank_points"`
	OverlapSections []string `json:"overlap_sections"`
	Fingerprint     string   `json:"fingerprint"`
}

type Planner struct {
	mu       sync.RWMutex
	topology Topology
}

func NewPlanner(topology Topology) *Planner {
	return &Planner{topology: cloneTopology(topology)}
}

func (p *Planner) Update(topology Topology) {
	p.mu.Lock()
	if topology.Revision <= p.topology.Revision {
		topology.Revision = p.topology.Revision + 1
	}
	p.topology = cloneTopology(topology)
	p.mu.Unlock()
}

func (p *Planner) Build(routeName string) Plan {
	p.mu.RLock()
	topology := cloneTopology(p.topology)
	p.mu.RUnlock()
	main := unique(topology.Main[routeName])
	flank := unique(topology.Flank[routeName])
	overlap := unique(topology.Overlap[routeName])
	fingerprint := planFingerprint(topology.Revision, main, flank, overlap)
	return Plan{RouteName: routeName, Revision: topology.Revision, MainSections: main, FlankPoints: flank, OverlapSections: overlap, Fingerprint: fingerprint}
}

func (p *Planner) Snapshot() Topology {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return cloneTopology(p.topology)
}

func planFingerprint(revision uint64, main, flank, overlap []string) string {
	parts := []string{string(rune(revision))}
	parts = append(parts, main...)
	parts = append(parts, flank...)
	parts = append(parts, overlap...)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}

func unique(values []string) []string {
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

func cloneTopology(input Topology) Topology {
	return Topology{Revision: input.Revision, Main: cloneMap(input.Main), Flank: cloneMap(input.Flank), Overlap: cloneMap(input.Overlap)}
}

func cloneMap(input map[string][]string) map[string][]string {
	result := make(map[string][]string, len(input))
	for key, values := range input {
		result[key] = append([]string(nil), values...)
	}
	return result
}

func (p Plan) Apply(route model.Route) model.Route {
	route.Sections = append([]string(nil), p.MainSections...)
	route.FlankPoints = append([]string(nil), p.FlankPoints...)
	route.OverlapSections = append([]string(nil), p.OverlapSections...)
	route.TopologyRevision = p.Revision
	return route
}
