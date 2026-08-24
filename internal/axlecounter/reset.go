package axlecounter

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ResetApproval struct {
	ID        string    `json:"id"`
	SectionID string    `json:"section_id"`
	OperatorA string    `json:"operator_a"`
	OperatorB string    `json:"operator_b"`
	At        time.Time `json:"at"`
}

type ResetResult struct {
	ApprovalID  string   `json:"approval_id"`
	SectionIDs  []string `json:"section_ids"`
	BoundaryIDs []string `json:"boundary_ids"`
	Committed   bool     `json:"committed"`
}

type ResetService struct {
	store   *Store
	mu      sync.Mutex
	history []ResetResult
}

func NewResetService(store *Store) *ResetService {
	return &ResetService{store: store}
}

func (s *ResetService) Approve(sectionID, operatorA, operatorB string, at time.Time) (ResetApproval, error) {
	if sectionID == "" || operatorA == "" || operatorB == "" || operatorA == operatorB {
		return ResetApproval{}, errors.New("two distinct operators must approve a section reset")
	}
	s.store.mu.RLock()
	_, exists := s.store.sections[sectionID]
	s.store.mu.RUnlock()
	if !exists {
		return ResetApproval{}, errors.New("unknown axle counter section")
	}
	return ResetApproval{ID: uuid.NewString(), SectionID: sectionID, OperatorA: operatorA, OperatorB: operatorB, At: at}, nil
}

func (s *ResetService) Commit(approvals []ResetApproval) (ResetResult, error) {
	if len(approvals) == 0 {
		return ResetResult{}, errors.New("reset approval is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	sections := make([]string, 0, len(approvals))
	boundarySet := map[string]struct{}{}
	for _, approval := range approvals {
		section, ok := s.store.sections[approval.SectionID]
		if !ok {
			return ResetResult{}, errors.New("approved section no longer exists")
		}
		sections = append(sections, section.ID)
		boundarySet[section.Head] = struct{}{}
		boundarySet[section.Tail] = struct{}{}
	}
	boundaries := make([]string, 0, len(boundarySet))
	for id := range boundarySet {
		boundary := s.store.boundaries[id]
		boundary.Baseline = boundary.Count
		boundary.Revision++
		boundary.UpdatedAt = time.Now().UTC()
		s.store.boundaries[id] = boundary
		boundaries = append(boundaries, id)
	}
	sort.Strings(sections)
	sort.Strings(boundaries)
	s.store.recomputeLocked()
	result := ResetResult{ApprovalID: approvals[0].ID, SectionIDs: sections, BoundaryIDs: boundaries, Committed: true}
	s.history = append(s.history, result)
	return result, nil
}

func (s *ResetService) CommitAdjacentConcurrently(first, second ResetApproval) (ResetResult, error) {
	resultChannel := make(chan ResetResult, 1)
	errorChannel := make(chan error, 1)
	go func() {
		result, err := s.Commit([]ResetApproval{first, second})
		if err != nil {
			errorChannel <- err
			return
		}
		resultChannel <- result
	}()
	select {
	case result := <-resultChannel:
		return result, nil
	case err := <-errorChannel:
		return ResetResult{}, err
	}
}

func (s *ResetService) History() []ResetResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]ResetResult(nil), s.history...)
}
