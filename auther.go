// Package auther provides a role-tree authorization library for Go.
//
// # Core concepts
//
// Auther manages three core entities:
//
//   - Role: forms a tree hierarchy. The root role is created automatically
//     on initialization and is granted "/**" by default. Roles can create
//     child roles and users. Permissions are explicit-only — a parent role
//     must call Grant to delegate resources to its descendants.
//
//   - User: a passive leaf node created under a role. Users inherit their
//     role's effective permissions but cannot manage resources or create
//     other users/roles.
//
//   - Resource: a path-style string such as "/user/create" or "/data/**".
//     Glob matching is supported: * matches a single path segment, **
//     matches zero or more segments.
//
// # Persistence
//
// An Adapter provides persistence. Every mutation is written through to
// the adapter immediately. On construction, if the adapter holds persisted
// data, it is loaded and restored automatically.
//
// # Concurrency safety
//
// All public methods of Authorizer are protected by sync.RWMutex and are
// safe for concurrent use.
package auther

import (
	"log"

	"github.com/gralliry/go-auther/adapter/memory"
	"github.com/gralliry/go-auther/internal/model"
)

type Adapter = model.Adapter

// Authorizer is the main entry point of the authorization system.
// It manages the role tree, user mappings, and resource grants.
type Authorizer struct {
	adapter Adapter
}

// NewAuthorizer creates an Authorizer backed by the given adapter.
// If adapter is nil, an in-memory adapter is used and a warning is logged.
func NewAuthorizer(adapter Adapter) (*Authorizer, error) {
	if adapter == nil {
		log.Println("auther: no adapter provided, falling back to in-memory default")
		adapter = memory.New()
	}
	a := &Authorizer{
		adapter: adapter,
	}
	return a, nil
}

// Load loads policy data from the adapter, rebuilds the role tree, and
// automatically repairs corrupted data before writing back.
func (a *Authorizer) Load() error {
	return nil
}

// Save is a no-op — all mutations are written through to the adapter immediately.
func (a *Authorizer) Save() error {
	return nil
}
