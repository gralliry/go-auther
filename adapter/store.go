// Package adapter defines the persistence interface for Auther.
//
// Implementations of the Store interface live in subdirectories
// (noop, json, sql, etc.) as independent Go modules.
package adapter

// Store defines the persistence interface for Auther.
//
// Implementations must be concurrency-safe.
//
// Design principles:
//   - Create methods are idempotent — duplicate records are silently ignored.
//   - User operations use a linking metaphor: every User record is a (user, role)
//     binding. A user without at least one role binding is not persisted.
type Store interface {
	Snapshot() (Snapshot, error)

	// Role methods.
	CreateRole(role Role) error
	DeleteRole(role Role) error

	// User methods — binding-based, not entity-based.
	// LinkUser creates a (user, role) binding; DeleteUser drops all bindings
	// for the user; UnlinkUser drops one specific binding.
	LinkUser(user User) error
	DeleteUser(user User) error
	UnlinkUser(user User) error

	// Policy methods.
	CreatePolicy(policy Policy) error
	DeletePolicy(policyID int64) error
}
