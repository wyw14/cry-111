package point

import (
	"errors"
	"time"

	"github.com/wyw14/cry-111/internal/model"
)

type Proof struct {
	PointID    string              `json:"point_id"`
	Position   model.PointPosition `json:"position"`
	Revision   uint64              `json:"revision"`
	ObservedAt time.Time           `json:"observed_at"`
}

type ProofReader struct {
	store *Store
}

func NewProofReader(store *Store) *ProofReader {
	return &ProofReader{store: store}
}

func (r *ProofReader) Read(id string) (Proof, error) {
	state, ok := r.store.Get(id)
	if !ok {
		return Proof{}, errors.New("unknown point")
	}
	if !state.PositionProved() {
		return Proof{}, errors.New("point does not have a current stable proof")
	}
	return Proof{PointID: id, Position: state.Detected, Revision: state.ProofRevision, ObservedAt: state.UpdatedAt}, nil
}

func (r *ProofReader) ReadAll(ids []string) ([]Proof, error) {
	proofs := make([]Proof, 0, len(ids))
	for _, id := range ids {
		proof, err := r.Read(id)
		if err != nil {
			return nil, err
		}
		proofs = append(proofs, proof)
	}
	return proofs, nil
}
