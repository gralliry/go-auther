// Package adapter defines the persistence interface for Auther.
//
// Implementations of the Adapter interface live in subdirectories
// (memory, json, sql, etc.) as independent Go modules.
//
// The interface uses only primitive Go types so that implementors
// have zero dependency on any shared structure definitions.
package adapter

// Adapter defines the persistence interface for Auther.
//
// Implementations must be concurrency-safe.
type Adapter interface {
	All() (Snapshot, error)

	// Role methods.
	CreateRole(role Role) error
	DeleteRole(role Role) error

	// User methods.
	CreateUser(user User) error
	DeleteUser(user User) error
	UnassignUser(user User) error

	// Policy methods.
	CreatePolicy(policy Policy) error
	DeletePolicy(policyID int64) error
}
