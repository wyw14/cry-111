package signal

import (
	"errors"
	"time"

	"github.com/wyw14/cry-111/internal/model"
)

type CurrentRange struct {
	Minimum int `json:"minimum"`
	Maximum int `json:"maximum"`
}

type LampProof struct {
	store  *Store
	ranges map[model.SignalAspect]CurrentRange
}

func NewLampProof(store *Store) *LampProof {
	return &LampProof{store: store, ranges: map[model.SignalAspect]CurrentRange{
		model.AspectStop:    {Minimum: 80, Maximum: 160},
		model.AspectCall:    {Minimum: 60, Maximum: 120},
		model.AspectProceed: {Minimum: 90, Maximum: 180},
		model.AspectDark:    {Minimum: 0, Maximum: 5},
	}}
}

func (p *LampProof) Evaluate(id string, at time.Time) (model.SignalState, error) {
	state, ok := p.store.Get(id)
	if !ok {
		return model.SignalState{}, errors.New("unknown signal")
	}
	rangeValue, ok := p.ranges[state.Commanded]
	if !ok {
		return model.SignalState{}, errors.New("unsupported signal aspect")
	}
	currentMatches := state.CurrentMilliAmps >= rangeValue.Minimum && state.CurrentMilliAmps <= rangeValue.Maximum
	selectionMatches := state.Selected == state.Commanded && state.Displayed == state.Commanded
	proved := currentMatches && selectionMatches
	dark := state.Commanded == model.AspectDark && proved
	return p.store.SetProof(id, proved, dark, at)
}

func (p *LampProof) Range(aspect model.SignalAspect) (CurrentRange, bool) {
	value, ok := p.ranges[aspect]
	return value, ok
}

func (p *LampProof) FailSafe(id string, at time.Time) (model.SignalState, error) {
	state, ok := p.store.Get(id)
	if !ok {
		return model.SignalState{}, errors.New("unknown signal")
	}
	if state.Commanded == model.AspectProceed || state.Commanded == model.AspectCall {
		_, err := p.store.Command(id, state.RouteID, model.AspectStop, at)
		if err != nil {
			return model.SignalState{}, err
		}
	}
	return p.store.SetProof(id, false, false, at)
}
