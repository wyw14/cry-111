package route

import (
	"errors"

	"github.com/wyw14/cry-111/internal/interlock"
	"github.com/wyw14/cry-111/internal/model"
	"github.com/wyw14/cry-111/internal/point"
)

type ProofBundle struct {
	RouteID          string        `json:"route_id"`
	Points           []point.Proof `json:"points"`
	TracksClear      bool          `json:"tracks_clear"`
	TopologyRevision uint64        `json:"topology_revision"`
}

type Prover struct {
	points    *point.ProofReader
	occupancy *interlock.OccupancyPermit
}

func NewProver(points *point.ProofReader, occupancy *interlock.OccupancyPermit) *Prover {
	return &Prover{points: points, occupancy: occupancy}
}

func (p *Prover) Prove(route model.Route) (ProofBundle, error) {
	if route.TopologyRevision == 0 {
		return ProofBundle{}, errors.New("route has no topology revision")
	}
	pointIDs := append(append([]string(nil), route.Points...), route.FlankPoints...)
	proofs, err := p.points.ReadAll(pointIDs)
	if err != nil {
		return ProofBundle{}, err
	}
	tracks := append(append([]string(nil), route.Sections...), route.OverlapSections...)
	if err := p.occupancy.RequireClear(tracks); err != nil {
		return ProofBundle{}, err
	}
	return ProofBundle{RouteID: route.ID, Points: proofs, TracksClear: true, TopologyRevision: route.TopologyRevision}, nil
}

func (p *Prover) Recheck(bundle ProofBundle) error {
	for _, proof := range bundle.Points {
		current, err := p.points.Read(proof.PointID)
		if err != nil || current.Revision != proof.Revision || current.Position != proof.Position {
			return errors.New("route point proof changed")
		}
	}
	return nil
}
