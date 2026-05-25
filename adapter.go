package auther

import "auther/snapshot"

// Adapter defines the persistence interface for Auther.
//
// All adapters must implement the full-snapshot methods (Load / Save).
// Incremental methods are used by the Authorizer for simple single-entity
// operations. For cascade operations (DeleteRole with descendants, Revoke
// with cascade), the Authorizer always calls Save().
//
// Implementations must be concurrency-safe.
type Adapter interface {
	// Load reads the complete policy snapshot from storage.
	// Return nil snapshot without error if no data exists yet.
	Load() (*snapshot.Policy, error)

	// Save persists the complete policy snapshot to storage.
	Save(s *snapshot.Policy) error

	// CreateRole persists a new role.
	CreateRole(role snapshot.Role) error

	// DeleteRole removes a single role.
	DeleteRole(role snapshot.Role) error

	// CreateUser persists a new user.
	CreateUser(user snapshot.User) error

	// DeleteUser removes a single user.
	DeleteUser(user snapshot.User) error

	// AddGrant persists a new grant.
	AddGrant(grant snapshot.Grant) error

	// RemoveGrant removes a single grant.
	RemoveGrant(grant snapshot.Grant) error
}
