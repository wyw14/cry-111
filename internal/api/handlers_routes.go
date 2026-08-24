package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/wyw14/cry-111/internal/flank"
)

func (s *Server) listRoutes(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"routes": s.app.Routes.List(), "definitions": s.app.Routes.Definitions()})
}

func (s *Server) requestRoute(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string `json:"name"`
	}
	if err := readJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	routeValue, err := s.app.Routes.Request(input.Name, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	for _, section := range append(append([]string(nil), routeValue.Sections...), routeValue.OverlapSections...) {
		if err := s.app.Tracks.SetReservation(section, routeValue.ID); err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
	}
	writeJSON(w, http.StatusCreated, routeValue)
}

func (s *Server) cancelRoute(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.Cancellation.Request(chi.URLParam(r, "routeID"), time.Now().UTC())
	if err != nil {
		writeJSON(w, http.StatusConflict, result)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) emergencyCancel(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TimeoutMillis int `json:"timeout_millis"`
	}
	if err := readJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if input.TimeoutMillis <= 0 {
		input.TimeoutMillis = 3000
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(input.TimeoutMillis)*time.Millisecond)
	defer cancel()
	result, err := s.app.Emergency.Request(ctx, chi.URLParam(r, "routeID"), time.Now().UTC())
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"result": result, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) confirmDark(w http.ResponseWriter, r *http.Request) {
	routeValue, ok := s.app.Routes.Get(chi.URLParam(r, "routeID"))
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("route not found"))
		return
	}
	if err := s.app.Emergency.ConfirmDark(routeValue.SignalID, time.Now().UTC()); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"signal_id": routeValue.SignalID, "dark_proved": true})
}

func (s *Server) updateTopology(w http.ResponseWriter, r *http.Request) {
	var input flank.Topology
	if err := readJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(input.Main) == 0 || len(input.Flank) == 0 || len(input.Overlap) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("topology must include main, flank and overlap maps"))
		return
	}
	s.app.FlankPlanner.Update(input)
	revision := s.app.FlankPlanner.Revision()
	// Any topology change must regenerate the complete safety plan — main path,
	// flank points and overlap sections — against the same revision. The flank
	// planner already carries main/flank/overlap for each route; the overlap
	// planner keeps a separate per-route plan that is validated against the
	// route's topology revision, so it must be reconfigured here as well.
	snapshot := s.app.FlankPlanner.Snapshot()
	for _, definition := range s.app.Routes.Definitions() {
		s.app.Resolver.Invalidate(definition.Name)
		s.app.OverlapPlans.Configure(definition.Name, snapshot.Overlap[definition.Name], revision)
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) revalidateRoute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "routeID")
	if err := s.app.Routes.Revalidate(id); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	if err := s.app.FlankProof.Verify(id); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	routeValue, _ := s.app.Routes.Get(id)
	writeJSON(w, http.StatusOK, map[string]any{"route_id": id, "phase": routeValue.Phase, "proof_valid": true})
}

func (s *Server) engageApproach(w http.ResponseWriter, r *http.Request) {
	var input struct {
		SectionID      string `json:"section_id"`
		DurationMillis int    `json:"duration_millis"`
	}
	if err := readJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if input.DurationMillis <= 0 {
		input.DurationMillis = 30000
	}
	id := chi.URLParam(r, "routeID")
	if err := s.app.Cancellation.EngageApproach(id, input.SectionID, time.Duration(input.DurationMillis)*time.Millisecond, time.Now().UTC()); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"route_id": id, "section_id": input.SectionID, "active": true})
}

func (s *Server) startPassage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "routeID")
	routeValue, ok := s.app.Routes.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("route not found"))
		return
	}
	var input struct {
		Direction string `json:"direction"`
	}
	if err := readJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, s.app.Sectional.Begin(id, routeValue.Sections, input.Direction, time.Now().UTC()))
}

func (s *Server) occupyPassage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "routeID")
	if err := s.app.Sectional.Occupy(id, chi.URLParam(r, "sectionID"), time.Now().UTC()); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"route_id": id, "accepted": true})
}

func (s *Server) clearPassage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "routeID")
	releaseValue, err := s.app.Sectional.ObserveClear(id, chi.URLParam(r, "sectionID"), time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"release": releaseValue, "history": s.app.Sectional.Releases(id)})
}

func (s *Server) completePassage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "routeID")
	s.app.Sectional.Complete(id)
	writeJSON(w, http.StatusOK, map[string]any{"route_id": id, "complete": true})
}
