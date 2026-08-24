package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/wyw14/cry-111/internal/model"
)

func (s *Server) listPoints(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"points": s.app.Points.List(), "stable_dwell": s.app.Detection.StableDwell().String()})
}

func (s *Server) pointProof(w http.ResponseWriter, r *http.Request) {
	state, err := s.app.Detection.RequireProof(chi.URLParam(r, "pointID"))
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"state": state, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (s *Server) commandPoint(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Position model.PointPosition `json:"position"`
	}
	if err := readJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if input.Position != model.PointNormal && input.Position != model.PointReverse {
		writeError(w, http.StatusBadRequest, errors.New("position must be normal or reverse"))
		return
	}
	state, err := s.app.Points.Command(chi.URLParam(r, "pointID"), input.Position, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusAccepted, state)
}

func (s *Server) detectPoint(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Position     model.PointPosition `json:"position"`
		Closed       bool                `json:"closed"`
		MotorRunning bool                `json:"motor_running"`
		ObservedAt   time.Time           `json:"observed_at"`
	}
	if err := readJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if input.ObservedAt.IsZero() {
		input.ObservedAt = time.Now().UTC()
	}
	state, err := s.app.Detection.Observe(chi.URLParam(r, "pointID"), input.Position, input.Closed, input.MotorRunning, input.ObservedAt)
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}
