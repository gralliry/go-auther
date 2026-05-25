// Package memoryadapter provides an in-memory adapter for testing and development.
package memoryadapter

import (
	"sync"

	"auther"
)

// MemoryAdapter stores policy snapshots in memory.
type MemoryAdapter struct {
	mu       sync.RWMutex
	snapshot *auther.PolicySnapshot
}

// NewMemoryAdapter creates a new in-memory adapter.
func NewMemoryAdapter() *MemoryAdapter {
	return &MemoryAdapter{}
}

// Load returns the stored policy snapshot, or nil if none exists.
func (a *MemoryAdapter) Load() (*auther.PolicySnapshot, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.snapshot == nil {
		return nil, nil
	}
	// Deep copy to avoid the caller mutating our stored state
	return copySnapshot(a.snapshot), nil
}

// Save persists a policy snapshot in memory.
func (a *MemoryAdapter) Save(snapshot *auther.PolicySnapshot) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.snapshot = copySnapshot(snapshot)
	return nil
}

// copySnapshot creates a deep copy of a PolicySnapshot.
func copySnapshot(s *auther.PolicySnapshot) *auther.PolicySnapshot {
	if s == nil {
		return nil
	}
	c := &auther.PolicySnapshot{}

	c.Roles = make([]auther.RoleSnapshot, len(s.Roles))
	copy(c.Roles, s.Roles)

	c.Users = make([]auther.UserSnapshot, len(s.Users))
	copy(c.Users, s.Users)

	c.Grants = make([]auther.GrantSnapshot, len(s.Grants))
	copy(c.Grants, s.Grants)

	return c
}
