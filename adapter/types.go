// Package adapter defines the persistence interface for Auther and the plain data
// types used by it. Implementations live in subdirectories as independent Go modules.
package adapter

// Role is a named container that receives and delegates resource access policies.
// Roles form a DAG via Grant/Revoke; permissions are explicit-only.
type Role struct {
	ID string
}

// User is a collection of role assignments. A user has access to a resource
// if any of their assigned roles has access via EnforceByUser.
// In the adapter, each User record represents one (user, role) binding pair —
// a user with multiple roles has multiple records.
type User struct {
	ID     string
	RoleID string
}

// Policy is a single resource grant from one role to another.
// It is identified by a snowflake ID and names the grantor, grantee, and resource pattern.
type Policy struct {
	ID int64

	GrantorRoleID string
	GranteeRoleID string

	Resource string
}

// Snapshot is a point-in-time copy of all persisted state.
// Returned by Store.Snapshot() at Manager construction time.
type Snapshot struct {
	Role   []Role
	User   []User
	Policy []Policy
}
