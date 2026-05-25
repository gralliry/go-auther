// Package memoryadapter provides an in-memory adapter for testing and development.
package memoryadapter

import (
	"sync"

	"auther/model"
)

// MemoryAdapter stores policy snapshots in memory.
type MemoryAdapter struct {
	mu       sync.RWMutex
	snapshot *model.PolicySnapshot
}

// NewMemoryAdapter creates a new in-memory adapter.
func NewMemoryAdapter() *MemoryAdapter {
	return &MemoryAdapter{}
}

// Load returns the stored policy snapshot, or nil if none exists.
func (a *MemoryAdapter) Load() (*model.PolicySnapshot, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.snapshot == nil {
		return nil, nil
	}
	// Deep copy to avoid the caller mutating our stored state
	return copySnapshot(a.snapshot), nil
}

// Save persists a policy snapshot in memory.
func (a *MemoryAdapter) Save(snapshot *model.PolicySnapshot) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.snapshot = copySnapshot(snapshot)
	return nil
}

// copySnapshot creates a deep copy of a PolicySnapshot.
func copySnapshot(s *model.PolicySnapshot) *model.PolicySnapshot {
	if s == nil {
		return nil
	}
	c := &model.PolicySnapshot{}

	c.Roles = make([]model.RoleSnapshot, len(s.Roles))
	for i, r := range s.Roles {
		c.Roles[i] = r
		c.Roles[i].Resources = make([]string, len(r.Resources))
		copy(c.Roles[i].Resources, r.Resources)
	}

	c.Users = make([]model.UserSnapshot, len(s.Users))
	copy(c.Users, s.Users)

	c.Grants = make([]model.GrantSnapshot, len(s.Grants))
	copy(c.Grants, s.Grants)

	return c
}
