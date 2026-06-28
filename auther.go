// Package auther is the public API surface of the role-tree authorization library.
// It re-exports the core types (Manager, Store) and the NewManager constructor
// from internal packages so callers only import this one package.
package auther

import (
	"github.com/gralliry/go-auther/adapter"
	"github.com/gralliry/go-auther/internal/manager"
)

// Public type aliases — the only types users need to reference directly.
type (
	// Manager is the top-level orchestrator for all role, user, and policy operations.
	Manager = manager.Manager
	// Store is the persistence interface that backend drivers must implement.
	Store = adapter.Store
)

// NewManager creates a Manager backed by the given Store, loading any
// previously persisted state.
var NewManager = manager.New
