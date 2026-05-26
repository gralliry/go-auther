// Package auther provides a role-tree authorization library for Go.
//
// Core concepts
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
// Persistence
//
// An Adapter provides persistence. Every mutation is written through to
// the adapter immediately. On construction, if the adapter holds persisted
// data, it is loaded and restored automatically.
//
// Concurrency safety
//
// All public methods of Authorizer are protected by sync.RWMutex and are
// safe for concurrent use.
package auther

import (
	"sync"

	"github.com/gralliry/go-auther/internal/model"
)

// Authorizer is the main entry point of the authorization system.
// It manages the role tree, user mappings, and resource grants.
type Authorizer struct {
	mu      sync.RWMutex
	roles   map[string]*model.RoleNode
	users   map[string]*model.UserNode
	adapter Adapter
}

// NewAuthorizer creates an Authorizer backed by the given adapter.
// If the adapter contains persisted data, it is loaded and restored.
// Otherwise a root role with ID "root" and resource "/**" is created automatically.
// adapter must not be nil.
func NewAuthorizer(adapter Adapter) (*Authorizer, error) {
	if adapter == nil {
		return nil, ErrAdapterRequired
	}
	a := &Authorizer{
		adapter: adapter,
		roles:   make(map[string]*model.RoleNode),
		users:   make(map[string]*model.UserNode),
	}
	if err := a.Load(); err != nil {
		return nil, err
	}
	if a.roles["root"] == nil {
		a.roles["root"] = &model.RoleNode{
			ID:         "root",
			Children:   make(map[string]*model.RoleNode),
			GrantedMap: map[string]bool{"/**": true},
			Users:      make(map[string]*model.UserNode),
		}
		if err := a.save(); err != nil {
			return nil, err
		}
	}
	return a, nil
}
