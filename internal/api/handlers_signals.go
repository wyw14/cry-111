package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/wyw14/cry-111/internal/model"
)

func (s *Server) listSignals(w http.ResponseWriter, _ *http.Request) {
	requested := map[string]model.SignalAspect{}
	for _, state := range s.app.Signals.List() {
		if aspect, ok := s.app.Selector.Requested(state.ID); ok {
			requested[state.ID] = aspect
		}
	}
	ranges := map[model.SignalAspect]any{}
	for _, aspect := range []model.SignalAspect{model.AspectStop, model.AspectCall, model.AspectProceed, model.AspectDark} {
		if currentRange, ok := s.app.LampProof.Range(aspect); ok {
			ranges[aspect] = currentRange
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"signals": s.app.Signals.List(), "requested": requested, "current_ranges": ranges})
}

func (s *Server) updateSignalCircuit(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Selected         model.SignalAspect `json:"selected"`
		Displayed        model.SignalAspect `json:"displayed"`
		CurrentMilliAmps int                `json:"current_milliamps"`
	}
	if err := readJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	id := chi.URLParam(r, "signalID")
	if _, err := s.app.Selector.Observe(id, input.Selected, input.Displayed, input.CurrentMilliAmps, time.Now().UTC()); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	state, err := s.app.LampProof.Evaluate(id, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (s *Server) stopSignal(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RouteID string `json:"route_id"`
	}
	if err := readJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	state, err := s.app.Signals.Stop(chi.URLParam(r, "signalID"), input.RouteID, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}
