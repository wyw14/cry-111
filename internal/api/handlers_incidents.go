package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func (s *Server) listIncidents(w http.ResponseWriter, _ *http.Request) {
	incidents := append(s.app.Occupancy.Incidents(), s.app.Power.Incidents()...)
	writeJSON(w, http.StatusOK, map[string]any{"incidents": incidents})
}

func (s *Server) ackIncident(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "incidentID")
	if !s.app.Occupancy.Acknowledge(id, time.Now().UTC()) {
		writeError(w, http.StatusNotFound, errors.New("active incident not found"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "acknowledged": true})
}
