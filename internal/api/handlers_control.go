package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/wyw14/cry-111/internal/model"
	"github.com/wyw14/cry-111/internal/power"
	"github.com/wyw14/cry-111/internal/route"
)

func (s *Server) listTracks(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"tracks": s.app.Tracks.List(), "all_clear": s.app.Tracks.AllClear([]string{"1DG", "3DG", "5DG", "7DG", "9DG", "11DG"})})
}

func (s *Server) updateTrack(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Occupied  bool   `json:"occupied"`
		Direction string `json:"direction"`
	}
	if err := readJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	state, err := s.app.Occupancy.Apply(chi.URLParam(r, "trackID"), input.Occupied, input.Direction, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (s *Server) resetAdjacentAxles(w http.ResponseWriter, r *http.Request) {
	var input struct {
		First     string `json:"first"`
		Second    string `json:"second"`
		OperatorA string `json:"operator_a"`
		OperatorB string `json:"operator_b"`
	}
	if err := readJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	first, err := s.app.AxleReset.Approve(input.First, input.OperatorA, input.OperatorB, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	second, err := s.app.AxleReset.Approve(input.Second, input.OperatorA, input.OperatorB, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := s.app.AxleReset.CommitAdjacentConcurrently(first, second)
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	_, sections := s.app.Axles.Snapshot()
	states := make([]model.TrackState, 0, len(result.SectionIDs))
	for _, sectionID := range result.SectionIDs {
		for _, section := range sections {
			if section.ID != sectionID {
				continue
			}
			state, applyErr := s.app.TrackSection.ApplyBaseline(section.ID, result.ApprovalID, section.Occupied, time.Now().UTC())
			if applyErr != nil {
				writeError(w, http.StatusConflict, applyErr)
				return
			}
			states = append(states, state)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"reset": result, "sections": states, "history": s.app.TrackSection.History()})
}

func (s *Server) listPower(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"domains": s.app.Domains.List(), "all_ready": s.app.Domains.AllReady()})
}

func (s *Server) updatePower(w http.ResponseWriter, r *http.Request) {
	var input struct {
		State power.DomainState `json:"state"`
	}
	if err := readJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	id := chi.URLParam(r, "domainID")
	var domain power.Domain
	var err error
	switch input.State {
	case power.DomainOffline:
		domain, err = s.app.Power.Lose(id, time.Now().UTC())
	case power.DomainSelfTesting:
		domain, err = s.app.Power.BeginRecovery(id, time.Now().UTC())
	case power.DomainReady:
		domain, err = s.app.Power.CompleteRecovery(id, time.Now().UTC())
	default:
		err = errors.New("unsupported power state")
	}
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, domain)
}

func (s *Server) correctClock(w http.ResponseWriter, r *http.Request) {
	var input struct {
		DeltaMillis int64 `json:"delta_millis"`
	}
	if err := readJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, s.app.Clock.Correct(time.Duration(input.DeltaMillis)*time.Millisecond))
}

func (s *Server) recoverRoutes(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Routes []model.Route `json:"routes"`
	}
	if err := readJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(input.Routes) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("at least one persisted route is required"))
		return
	}
	if err := s.app.Recovery.RestoreRoutes(input.Routes); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	results := make([]any, 0, len(input.Routes))
	for _, routeValue := range input.Routes {
		if result, ok := s.app.Reprover.Result(routeValue.ID); ok {
			results = append(results, result)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"restored": len(input.Routes), "phase": model.RouteProving, "results": results})
}

func (s *Server) crossingState(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"gate": s.app.Gate.Snapshot(), "sessions": s.app.Crossing.List()})
}

func (s *Server) closeCrossing(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RouteID string `json:"route_id"`
	}
	if err := readJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	session, err := s.app.Crossing.Close(input.RouteID, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusAccepted, session)
}

func (s *Server) proveCrossingDown(w http.ResponseWriter, r *http.Request) {
	session, err := s.app.Crossing.ProveDown(chi.URLParam(r, "sessionID"), time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (s *Server) recloseCrossing(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RouteID string `json:"route_id"`
	}
	if err := readJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	session, err := s.app.Crossing.Reclose(chi.URLParam(r, "sessionID"), input.RouteID, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session": session, "protected": s.app.Crossing.Protected(session.ID)})
}

func (s *Server) interlockStatus(w http.ResponseWriter, _ *http.Request) {
	routes := s.app.Routes.List()
	lockedPoints := map[string][]string{}
	signalled := 0
	for _, routeValue := range routes {
		lockedPoints[routeValue.ID] = s.app.PointLocks.Locked(routeValue.ID)
		if route.IsAtLeast(routeValue, model.RouteSignalled) {
			signalled++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"resource_count": s.app.Resources.Count(),
		"decisions":      s.app.Interlocking.Decisions(),
		"power_ready":    s.app.Domains.AllReady(),
		"route_count":    len(routes),
		"signalled":      signalled,
		"locked_points":  lockedPoints,
		"track_1dg_safe": s.app.Resources.Count() == 0 || s.app.Tracks.AllClear([]string{"1DG"}),
		"point_count":    len(s.app.Points.List()),
		"signal_count":   len(s.app.Signals.List()),
		"track_count":    len(s.app.Tracks.List()),
		"overlap_plans":  s.app.OverlapPlans.List(),
	})
}
