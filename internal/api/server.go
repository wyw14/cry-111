package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	app       *Application
	router    chi.Router
	startedAt time.Time
}

func NewServer(app *Application) *Server {
	server := &Server{app: app, router: chi.NewRouter(), startedAt: time.Now().UTC()}
	server.routes()
	return server
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) routes() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Recoverer)
	s.router.Get("/healthz", s.health)
	s.router.Get("/routes", s.routesPage)
	s.router.Get("/points", s.pointsPage)
	s.router.Get("/signals", s.signalsPage)
	s.router.Get("/incidents", s.incidentsPage)
	s.router.Get("/assets/app.css", s.stylesheet)
	s.router.Get("/assets/app.js", s.javascript)
	s.router.Route("/api", func(r chi.Router) {
		r.Get("/routes", s.listRoutes)
		r.Post("/routes/request", s.requestRoute)
		r.Post("/routes/{routeID}/cancel", s.cancelRoute)
		r.Post("/routes/{routeID}/emergency", s.emergencyCancel)
		r.Post("/routes/{routeID}/dark", s.confirmDark)
		r.Post("/routes/{routeID}/revalidate", s.revalidateRoute)
		r.Post("/routes/{routeID}/approach", s.engageApproach)
		r.Post("/routes/{routeID}/passage", s.startPassage)
		r.Post("/routes/{routeID}/passage/{sectionID}/occupied", s.occupyPassage)
		r.Post("/routes/{routeID}/passage/{sectionID}/clear", s.clearPassage)
		r.Delete("/routes/{routeID}/passage", s.completePassage)
		r.Put("/topology", s.updateTopology)
		r.Get("/points", s.listPoints)
		r.Post("/points/{pointID}/command", s.commandPoint)
		r.Post("/points/{pointID}/detection", s.detectPoint)
		r.Get("/points/{pointID}/proof", s.pointProof)
		r.Get("/signals", s.listSignals)
		r.Post("/signals/{signalID}/circuit", s.updateSignalCircuit)
		r.Post("/signals/{signalID}/stop", s.stopSignal)
		r.Get("/tracks", s.listTracks)
		r.Post("/tracks/{trackID}/occupancy", s.updateTrack)
		r.Get("/incidents", s.listIncidents)
		r.Post("/incidents/{incidentID}/ack", s.ackIncident)
		r.Post("/axle/resets/adjacent", s.resetAdjacentAxles)
		r.Get("/power", s.listPower)
		r.Post("/power/{domainID}/state", s.updatePower)
		r.Post("/clock/correct", s.correctClock)
		r.Post("/recovery/routes", s.recoverRoutes)
		r.Get("/crossing", s.crossingState)
		r.Post("/crossing/close", s.closeCrossing)
		r.Post("/crossing/{sessionID}/down", s.proveCrossingDown)
		r.Post("/crossing/{sessionID}/reclose", s.recloseCrossing)
		r.Get("/interlock/status", s.interlockStatus)
	})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "started_at": s.startedAt, "power_ready": s.app.Domains.AllReady()})
}

func readJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error()})
}
