// Package snapshot defines the flat serializable types used by Adapter for persistence.
package snapshot

// Policy is a complete flat snapshot of policy state.
type Policy struct {
	Roles  []Role  `json:"roles"`
	Users  []User  `json:"users"`
	Grants []Grant `json:"grants"`
}

// Role is a flat role record for serialization.
type Role struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id"`
}

// User is a flat user record for serialization.
type User struct {
	ID     string `json:"id"`
	RoleID string `json:"role_id"`
}

// Grant is a flat grant record for serialization.
type Grant struct {
	FromRoleID string `json:"from_role_id"`
	ToRoleID   string `json:"to_role_id"`
	Resource   string `json:"resource"`
}
