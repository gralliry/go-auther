// Package adapter defines the persistence interface for Auther.
//
// Implementations of the Adapter interface live in subdirectories
// (memory, json, sql, etc.) as independent Go modules.
//
// The interface uses only primitive Go types so that implementors
// have zero dependency on any shared structure definitions.
package model

// Adapter defines the persistence interface for Auther.
//
// Implementations must be concurrency-safe.
type Adapter interface {
	// Role methods.

	// AllRoles returns [][roleID, parentID].
	CreateRole(roleID, parentID string) error
	DeleteRole(roleID string) error
	// roleID parentID
	AllRoles() ([][2]string, error)

	// User methods.
	// AllUsers returns [][userID, roleID].
	CreateUser(roleID, userID string) error
	DeleteUser(userID string) error
	// userID roleID
	AllUsers() ([][2]string, error)

	// Grant methods.
	// AllGrants returns [][srcRoleID, dstRoleID, resource].
	CreateGrant(srcRoleID, dstRoleID, resource string) error
	DeleteGrant(srcRoleID, dstRoleID, resource string) error
	// grantor grantee resource
	AllGrants() ([][3]string, error)
}
