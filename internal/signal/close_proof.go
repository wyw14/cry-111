package signal

import (
	"context"
	"errors"
	"time"

	"github.com/wyw14/cry-111/internal/model"
)

type CloseProof struct {
	selector     *Selector
	lamp         *LampProof
	pollInterval time.Duration
}

func NewCloseProof(selector *Selector, lamp *LampProof, pollInterval time.Duration) *CloseProof {
	return &CloseProof{selector: selector, lamp: lamp, pollInterval: pollInterval}
}

func (p *CloseProof) Request(id, routeID string, at time.Time) error {
	_, err := p.selector.Command(id, routeID, model.AspectDark, at)
	return err
}

func (p *CloseProof) ObserveDark(id string, at time.Time) (model.SignalState, error) {
	_, err := p.selector.Observe(id, model.AspectDark, model.AspectDark, 0, at)
	if err != nil {
		return model.SignalState{}, err
	}
	return p.lamp.Evaluate(id, at)
}

func (p *CloseProof) Wait(ctx context.Context, id string) error {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()
	for {
		state, ok := p.lamp.store.Get(id)
		if !ok {
			return errors.New("unknown signal")
		}
		if state.DarkProved {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (p *CloseProof) Proved(id string) bool {
	state, ok := p.lamp.store.Get(id)
	return ok && state.DarkProved
}
