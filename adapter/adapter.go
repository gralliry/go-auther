// Package adapter defines the persistence interface for Auther.
//
// Implementations of the Adapter interface live in subdirectories
// (memory, json, sql, etc.) as independent Go modules.
//
// The interface and its entity types are separated: this package defines
// the contract; github.com/gralliry/go-auther/entity provides the types.
package adapter

import "github.com/gralliry/go-auther/entity"

// Adapter defines the persistence interface for Auther.
//
// Implementations must be concurrency-safe.
//
// Design principles:
//   - Create methods are idempotent — duplicate records are silently ignored.
//   - User operations use a linking metaphor: every User record is a (user, role)
//     binding. A user without at least one role binding is not persisted.
type Adapter interface {
	Snapshot() (entity.Snapshot, error)

	// Role methods.
	CreateRole(role entity.Role) error
	DeleteRole(role entity.Role) error

	// User methods — binding-based, not entity-based.
	// LinkUser creates a (user, role) binding; DeleteUser drops all bindings
	// for the user; UnlinkUser drops one specific binding.
	LinkUser(user entity.User) error
	DeleteUser(user entity.User) error
	UnlinkUser(user entity.User) error

	// Policy methods.
	CreatePolicy(policy entity.Policy) error
	DeletePolicy(policyID int64) error
}
