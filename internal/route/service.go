package route

import (
	"errors"
	"sync"
	"time"

	"github.com/wyw14/cry-111/internal/model"
	"github.com/wyw14/cry-111/internal/signal"
)

type Definition struct {
	Name     string   `json:"name"`
	Kind     string   `json:"kind"`
	SignalID string   `json:"signal_id"`
	Points   []string `json:"points"`
}

type Service struct {
	store       *Store
	resolver    *Resolver
	prover      *Prover
	transaction *Transaction
	permits     *signal.PermitService
	aspects     *signal.AspectService
	mu          sync.RWMutex
	definitions map[string]Definition
	proofs      map[string]ProofBundle
}

func NewService(store *Store, resolver *Resolver, prover *Prover, transaction *Transaction, permits *signal.PermitService, aspects *signal.AspectService) *Service {
	return &Service{store: store, resolver: resolver, prover: prover, transaction: transaction, permits: permits, aspects: aspects, definitions: map[string]Definition{}, proofs: map[string]ProofBundle{}}
}

func (s *Service) Define(definition Definition) error {
	if definition.Name == "" || definition.SignalID == "" || len(definition.Points) == 0 {
		return errors.New("route definition requires name, signal and points")
	}
	definition.Points = append([]string(nil), definition.Points...)
	s.mu.Lock()
	s.definitions[definition.Name] = definition
	s.mu.Unlock()
	return nil
}

func (s *Service) Request(name string, at time.Time) (model.Route, error) {
	s.mu.RLock()
	definition, ok := s.definitions[name]
	s.mu.RUnlock()
	if !ok {
		return model.Route{}, errors.New("route definition not found")
	}
	route, err := s.store.Create(definition.Name, definition.Kind, definition.SignalID, at)
	if err != nil {
		return model.Route{}, err
	}
	route.Points = append([]string(nil), definition.Points...)
	route = s.resolver.Resolve(route)
	if err := s.store.Put(route); err != nil {
		return model.Route{}, err
	}
	if _, err := s.store.Transition(route.ID, model.RouteProving, ""); err != nil {
		return model.Route{}, err
	}
	proof, err := s.prover.Prove(route)
	if err != nil {
		s.store.Transition(route.ID, model.RouteCancelled, err.Error())
		return model.Route{}, err
	}
	transactionResult, err := s.transaction.Commit(route)
	if err != nil {
		s.store.Transition(route.ID, model.RouteCancelled, err.Error())
		return model.Route{}, err
	}
	s.mu.Lock()
	s.proofs[route.ID] = proof
	s.mu.Unlock()
	locked, err := s.store.Transition(route.ID, model.RouteLocked, "")
	if err != nil {
		s.transaction.Rollback(route.ID)
		return model.Route{}, err
	}
	permit := signal.PermitSnapshot{RouteID: route.ID, MainLocked: len(transactionResult.Reservation.Resources) > 0, FlankLocked: len(route.FlankPoints) == len(transactionResult.Flank), OverlapLocked: len(route.OverlapSections) == len(transactionResult.Overlap.Sections), CrossingProtected: true, PowerReady: true}
	if err := s.permits.Publish(route.SignalID, permit); err != nil {
		s.transaction.Rollback(route.ID)
		return model.Route{}, err
	}
	if err := s.permits.Check(route.SignalID, route.ID); err != nil {
		s.transaction.Rollback(route.ID)
		return model.Route{}, err
	}
	if _, err := s.aspects.Open(route.SignalID, route.ID, at); err != nil {
		s.transaction.Rollback(route.ID)
		return model.Route{}, err
	}
	return s.store.Transition(locked.ID, model.RouteSignalled, "")
}

func (s *Service) Revalidate(routeID string) error {
	s.mu.RLock()
	proof, ok := s.proofs[routeID]
	s.mu.RUnlock()
	if !ok {
		return errors.New("route proof not found")
	}
	return s.prover.Recheck(proof)
}

func (s *Service) Get(id string) (model.Route, bool) {
	return s.store.Get(id)
}

func (s *Service) List() []model.Route {
	return s.store.List()
}

func (s *Service) Definitions() []Definition {
	s.mu.RLock()
	items := make([]Definition, 0, len(s.definitions))
	for _, definition := range s.definitions {
		definition.Points = append([]string(nil), definition.Points...)
		items = append(items, definition)
	}
	s.mu.RUnlock()
	return items
}

func (s *Service) Store() *Store {
	return s.store
}
