package signal

import (
	"errors"
	"time"

	"github.com/wyw14/cry-111/internal/model"
)

type AspectService struct {
	selector *Selector
	lamp     *LampProof
}

func NewAspectService(selector *Selector, lamp *LampProof) *AspectService {
	return &AspectService{selector: selector, lamp: lamp}
}

func (s *AspectService) Open(id, routeID string, at time.Time) (model.SignalState, error) {
	state, err := s.selector.Command(id, routeID, model.AspectProceed, at)
	if err != nil {
		return model.SignalState{}, err
	}
	_, err = s.selector.Observe(id, model.AspectProceed, model.AspectProceed, 120, at)
	if err != nil {
		return model.SignalState{}, err
	}
	proved, err := s.lamp.Evaluate(id, at)
	if err != nil {
		return model.SignalState{}, err
	}
	if !proved.AspectProved() {
		return state, errors.New("proceed aspect was not proved")
	}
	return proved, nil
}

func (s *AspectService) Stop(id, routeID string, at time.Time) (model.SignalState, error) {
	_, err := s.selector.Command(id, routeID, model.AspectStop, at)
	if err != nil {
		return model.SignalState{}, err
	}
	_, err = s.selector.Observe(id, model.AspectStop, model.AspectStop, 110, at)
	if err != nil {
		return model.SignalState{}, err
	}
	state, err := s.lamp.Evaluate(id, at)
	if err == nil {
		s.selector.Clear(id)
	}
	return state, err
}

func (s *AspectService) State(id string) (model.SignalState, bool) {
	return s.lamp.store.Get(id)
}

func (s *AspectService) List() []model.SignalState {
	return s.lamp.store.List()
}
