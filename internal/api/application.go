package api

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/wyw14/cry-111/internal/approach"
	"github.com/wyw14/cry-111/internal/axlecounter"
	stationclock "github.com/wyw14/cry-111/internal/clock"
	"github.com/wyw14/cry-111/internal/crossing"
	"github.com/wyw14/cry-111/internal/flank"
	"github.com/wyw14/cry-111/internal/interlock"
	"github.com/wyw14/cry-111/internal/journal"
	"github.com/wyw14/cry-111/internal/model"
	"github.com/wyw14/cry-111/internal/overlap"
	"github.com/wyw14/cry-111/internal/point"
	"github.com/wyw14/cry-111/internal/power"
	"github.com/wyw14/cry-111/internal/release"
	"github.com/wyw14/cry-111/internal/route"
	"github.com/wyw14/cry-111/internal/signal"
	"github.com/wyw14/cry-111/internal/track"
)

type Application struct {
	Routes       *route.Service
	RouteStore   *route.Store
	Resolver     *route.Resolver
	Cancellation *route.CancellationService
	Emergency    *route.EmergencyCanceller
	Tracks       *track.Store
	TrackSection *track.Section
	Occupancy    *track.OccupancyService
	Points       *point.Store
	Detection    *point.DetectionService
	PointLocks   *point.LockManager
	Signals      *signal.AspectService
	SignalStore  *signal.Store
	Selector     *signal.Selector
	LampProof    *signal.LampProof
	CloseProof   *signal.CloseProof
	Permits      *signal.PermitService
	Axles        *axlecounter.Store
	AxleReset    *axlecounter.ResetService
	Crossing     *crossing.Controller
	Gate         *crossing.Gate
	Power        *power.Service
	Domains      *power.DomainSet
	Clock        *stationclock.Service
	Journal      *journal.Store
	Snapshots    *journal.SnapshotStore
	Recovery     *journal.Recovery
	Reprover     *route.Reprover
	Interlocking *interlock.Engine
	Resources    *interlock.Resources
	Sectional    *release.Sectional
	Release      *release.Resources
	FlankPlanner *flank.Planner
	FlankProof   *flank.ProtectionService
	OverlapPlans *overlap.Planner
}

func NewApplication(dataDir string) (*Application, error) {
	now := time.Now().UTC()
	trackStore := track.NewStore([]string{"1DG", "3DG", "5DG", "7DG", "9DG", "11DG", "OV14", "OV22", "XING1", "12AC", "13AC", "14AC"})
	trackSection := track.NewSection(trackStore)
	pointStore := point.NewStore([]string{"P18", "P21", "P24", "P31", "P37", "P42"})
	signalStore := signal.NewStore([]string{"S8", "S14", "S22", "SH37"})
	detection := point.NewDetectionService(pointStore, 400*time.Millisecond)
	proofReader := point.NewProofReader(pointStore)
	pointLocks := point.NewLockManager(pointStore)
	topology := defaultTopology()
	flankPlanner := flank.NewPlanner(topology)
	flankService := flank.NewProtectionService(proofReader, pointLocks)
	overlapPlanner := overlap.NewPlanner()
	for routeName, sections := range topology.Overlap {
		overlapPlanner.Configure(routeName, sections, topology.Revision)
		if _, err := overlapPlanner.Resolve(routeName, topology.Revision); err != nil {
			return nil, err
		}
	}
	overlapStore := overlap.NewReservationStore()
	resources := interlock.NewResources()
	occupancyPermit := interlock.NewOccupancyPermit()
	for _, state := range trackStore.List() {
		occupancyPermit.Update(state)
	}
	trackStore.RegisterWatcher(occupancyPermit.Update)
	engine := interlock.NewEngine(resources, occupancyPermit)
	selector := signal.NewSelector(signalStore)
	lampProof := signal.NewLampProof(signalStore)
	aspects := signal.NewAspectService(selector, lampProof)
	closeProof := signal.NewCloseProof(selector, lampProof, 10*time.Millisecond)
	permits := signal.NewPermitService()
	resolver := route.NewResolver(flankPlanner)
	prover := route.NewProver(proofReader, occupancyPermit)
	transaction := route.NewTransaction(engine, flankService, overlapStore)
	routeStore := route.NewStore()
	routeService := route.NewService(routeStore, resolver, prover, transaction, permits, aspects)
	for _, definition := range defaultRouteDefinitions() {
		if err := routeService.Define(definition); err != nil {
			return nil, err
		}
	}
	stationClock := stationclock.New(now)
	approachLocks := approach.NewLockingService()
	approachTimer := approach.NewTimer(stationClock)
	cancellation := route.NewCancellationService(routeStore, approachLocks, approachTimer, transaction)
	passageTracker := track.NewPassageTracker(300 * time.Millisecond)
	sectional := release.NewSectional(passageTracker, trackStore)
	releaseResources := release.NewResources(engine, flankService, overlapStore, closeProof)
	emergency := route.NewEmergencyCanceller(routeStore, closeProof, releaseResources)
	axleStore := axlecounter.NewStore()
	if err := configureAxles(axleStore); err != nil {
		return nil, err
	}
	axleReset := axlecounter.NewResetService(axleStore)
	gate := crossing.NewGate("LC1")
	crossingController := crossing.NewController(gate)
	domains := power.NewDomainSet([]string{"logic", "point-relay", "signal-lamp", "track-circuit"})
	powerService := power.NewService(domains)
	if err := powerService.StartAll(now); err != nil {
		return nil, err
	}
	reprover := route.NewReprover(domains, routeStore, prover, []string{"logic", "point-relay", "signal-lamp", "track-circuit"})
	recovery := journal.NewRecovery(domains, reprover, reprover.RequiredDomains())
	journalStore, err := journal.NewStore(filepath.Join(dataDir, "events.jsonl"))
	if err != nil {
		return nil, err
	}
	snapshotStore, err := journal.NewSnapshotStore(filepath.Join(dataDir, "snapshot.json"))
	if err != nil {
		return nil, err
	}
	occupancy := track.NewOccupancyService(trackStore)
	app := &Application{Routes: routeService, RouteStore: routeStore, Resolver: resolver, Cancellation: cancellation, Emergency: emergency, Tracks: trackStore, TrackSection: trackSection, Occupancy: occupancy, Points: pointStore, Detection: detection, PointLocks: pointLocks, Signals: aspects, SignalStore: signalStore, Selector: selector, LampProof: lampProof, CloseProof: closeProof, Permits: permits, Axles: axleStore, AxleReset: axleReset, Crossing: crossingController, Gate: gate, Power: powerService, Domains: domains, Clock: stationClock, Journal: journalStore, Snapshots: snapshotStore, Recovery: recovery, Reprover: reprover, Interlocking: engine, Resources: resources, Sectional: sectional, Release: releaseResources, FlankPlanner: flankPlanner, FlankProof: flankService, OverlapPlans: overlapPlanner}
	if err := app.recordStartup(now); err != nil {
		return nil, err
	}
	return app, nil
}

func (a *Application) recordStartup(at time.Time) error {
	event := model.NewEvent("system.started", "signalroute", map[string]any{"domains": len(a.Domains.List()), "routes": len(a.Routes.Definitions())})
	event.At = at
	identity, err := model.ParseIdentity(event.ID)
	if err != nil {
		return err
	}
	if identity.Empty() {
		return errors.New("startup event identity is empty")
	}
	stored, err := a.Journal.Append(event)
	if err != nil {
		return err
	}
	if _, err := a.Snapshots.Save(stored.Sequence, map[string]any{"routes": a.Routes.List(), "power": a.Domains.List()}, at); err != nil {
		return err
	}
	_, err = a.Snapshots.Load()
	return err
}

func defaultTopology() flank.Topology {
	return flank.Topology{
		Revision: 1,
		Main:     map[string][]string{"R14": {"1DG", "3DG", "5DG"}, "R22": {"7DG", "9DG", "11DG"}, "SH37": {"3DG", "OV14"}},
		Flank:    map[string][]string{"R14": {"P37", "P42"}, "R22": {"P24", "P31"}, "SH37": {"P18"}},
		Overlap:  map[string][]string{"R14": {"OV14"}, "R22": {"OV22"}, "SH37": {"5DG"}},
	}
}

func defaultRouteDefinitions() []route.Definition {
	return []route.Definition{
		{Name: "R14", Kind: "train", SignalID: "S14", Points: []string{"P18", "P21"}},
		{Name: "R22", Kind: "train", SignalID: "S22", Points: []string{"P24", "P31"}},
		{Name: "SH37", Kind: "shunt", SignalID: "SH37", Points: []string{"P37"}},
	}
}

func configureAxles(store *axlecounter.Store) error {
	if err := store.ConfigureSection("12AC", "B11", "B12"); err != nil {
		return err
	}
	if err := store.ConfigureSection("13AC", "B12", "B13"); err != nil {
		return err
	}
	return store.ConfigureSection("14AC", "B13", "B14")
}
