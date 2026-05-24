package auther

import "errors"

var (
	// ErrUserNotFound is returned when a user lookup fails.
	ErrUserNotFound = errors.New("auther: user not found")

	// ErrRoleNotFound is returned when a role lookup fails.
	ErrRoleNotFound = errors.New("auther: role not found")

	// ErrGrantNotFound is returned when a grant lookup fails.
	ErrGrantNotFound = errors.New("auther: grant not found")

	// ErrNotAncestor is returned when a grant is attempted between roles
	// that are not in an ancestor-descendant relationship.
	ErrNotAncestor = errors.New("auther: grant target must be a descendant of the grantor role")

	// ErrCircularRoleHierarchy is returned when a role hierarchy loop is detected.
	ErrCircularRoleHierarchy = errors.New("auther: circular role hierarchy detected")

	// ErrInvalidResource is returned when a resource pattern is invalid.
	ErrInvalidResource = errors.New("auther: invalid resource pattern")

	// ErrDuplicateUser is returned when adding a user that already exists.
	ErrDuplicateUser = errors.New("auther: user already exists")

	// ErrDuplicateRole is returned when adding a role that already exists.
	ErrDuplicateRole = errors.New("auther: role already exists")

	// ErrDuplicateGrant is returned when adding a grant that already exists.
	ErrDuplicateGrant = errors.New("auther: grant already exists")

	// ErrRootRoleDelete is returned when attempting to delete the root role.
	ErrRootRoleDelete = errors.New("auther: cannot delete root role")

	// ErrRoleHasSubRoles is returned when deleting a role that still has child roles.
	ErrRoleHasSubRoles = errors.New("auther: cannot delete role with sub-roles; delete sub-roles first")

	// ErrRoleHasUsers is returned when deleting a role that still has users.
	ErrRoleHasUsers = errors.New("auther: cannot delete role with users; delete users first")
)
