package main

import "errors"

// Sentinel errors. Callers can test with errors.Is.
var (
	// ErrAdapterRequired is returned when a nil adapter is provided.
	ErrAdapterRequired = errors.New("auther: adapter is required")

	// ErrUserNotFound is returned when the specified user does not exist.
	ErrUserNotFound = errors.New("auther: user not found")

	// ErrRoleNotFound is returned when the specified role does not exist.
	ErrRoleNotFound = errors.New("auther: role not found")

	// ErrGrantNotFound is returned when the specified grant does not exist.
	ErrGrantNotFound = errors.New("auther: grant not found")

	// ErrNotAncestor is returned when the grantor is not an ancestor of the grantee.
	ErrNotAncestor = errors.New("auther: grant target must be a descendant of the grantor role")

	// ErrCircularRoleHierarchy is returned when a cycle is detected in the role tree.
	ErrCircularRoleHierarchy = errors.New("auther: circular role hierarchy detected")

	// ErrInvalidResource is returned when a resource pattern is invalid.
	ErrInvalidResource = errors.New("auther: invalid resource pattern")

	// ErrDuplicateUser is returned when attempting to create an already existing user.
	ErrDuplicateUser = errors.New("auther: user already exists")

	// ErrDuplicateRole is returned when attempting to create an already existing role.
	ErrDuplicateRole = errors.New("auther: role already exists")

	// ErrDuplicateGrant is returned when attempting to create an already existing grant.
	ErrDuplicateGrant = errors.New("auther: grant already exists")

	// ErrRootRoleDelete is returned when attempting to delete the root role.
	ErrRootRoleDelete = errors.New("auther: cannot delete root role")
)
