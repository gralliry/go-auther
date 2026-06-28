// Package auther is the public API surface of the role-tree authorization library.
// It re-exports the core types (Manager, Adapter) and the NewManager constructor
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
	// Adapter is the persistence interface that backend drivers must implement.
	Adapter = adapter.Adapter
)

// NewManager creates a Manager backed by the given Adapter, loading any
// previously persisted state.
var NewManager = manager.New
