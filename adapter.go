package auther

import "github.com/gralliry/go-auther/snapshot"

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

	// SetRole persists a new role.
	SetRole(role snapshot.Role) error

	// UnsetRole removes a single role.
	UnsetRole(role snapshot.Role) error

	// SetUser persists a new user.
	SetUser(user snapshot.User) error

	// UnsetUser removes a single user.
	UnsetUser(user snapshot.User) error

	// SetGrant persists a new grant.
	SetGrant(grant snapshot.Grant) error

	// UnsetGrant removes a single grant.
	UnsetGrant(grant snapshot.Grant) error
}
