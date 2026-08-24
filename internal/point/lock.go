package point

import (
	"errors"
	"sort"
	"sync"
)

type LockManager struct {
	store   *Store
	mu      sync.Mutex
	byRoute map[string][]string
}

func NewLockManager(store *Store) *LockManager {
	return &LockManager{store: store, byRoute: map[string][]string{}}
}

func (m *LockManager) Lock(routeID string, pointIDs []string) error {
	ids := uniqueSorted(pointIDs)
	m.mu.Lock()
	defer m.mu.Unlock()
	locked := make([]string, 0, len(ids))
	for _, id := range ids {
		state, ok := m.store.Get(id)
		if !ok || !state.PositionProved() {
			m.rollback(routeID, locked)
			return errors.New("point proof unavailable")
		}
		if err := m.store.SetLock(id, routeID); err != nil {
			m.rollback(routeID, locked)
			return err
		}
		locked = append(locked, id)
	}
	m.byRoute[routeID] = ids
	return nil
}

func (m *LockManager) Unlock(routeID string) []string {
	m.mu.Lock()
	ids := append([]string(nil), m.byRoute[routeID]...)
	delete(m.byRoute, routeID)
	for _, id := range ids {
		m.store.ClearLock(id, routeID)
	}
	m.mu.Unlock()
	return ids
}

func (m *LockManager) Locked(routeID string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.byRoute[routeID]...)
}

func (m *LockManager) rollback(routeID string, ids []string) {
	for _, id := range ids {
		m.store.ClearLock(id, routeID)
	}
}

func uniqueSorted(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		if value != "" {
			set[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
