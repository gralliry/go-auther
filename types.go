package auther

// =============================================================================
// Core data model
// =============================================================================

// RoleNode represents a role in the role tree.
type RoleNode struct {
	ID        string
	Parent    *RoleNode
	Children  map[string]*RoleNode
	Resources map[string]bool
	GrantsIn  []RoleGrant
	GrantsOut []RoleGrant
	Users     map[string]*UserNode
}

// UserNode represents a user in the authorization system.
type UserNode struct {
	ID   string
	Role *RoleNode
}

// RoleGrant represents an explicit resource delegation from one role to a descendant.
type RoleGrant struct {
	FromRoleID string
	ToRoleID   string
	Resource   string
}

// RoleInfo is the public view of a role, returned by GetRole/GetAllRoles.
type RoleInfo struct {
	ID         string
	ParentID   string
	Resources  []string
	SubRoleIDs []string
	UserIDs    []string
	GrantsIn   []RoleGrant
	GrantsOut  []RoleGrant
}

// UserInfo is the public view of a user, returned by GetUser/GetAllUsers.
type UserInfo struct {
	ID     string
	RoleID string
}

// =============================================================================
// Snapshot DTOs for adapter serialization
// =============================================================================

// PolicySnapshot is the flat, JSON-serializable representation of the entire
// authorization state. Used by adapters for persistence.
type PolicySnapshot struct {
	Roles  []RoleSnapshot  `json:"roles"`
	Users  []UserSnapshot  `json:"users"`
	Grants []GrantSnapshot `json:"grants"`
}

// RoleSnapshot is a flat role record for serialization.
type RoleSnapshot struct {
	ID        string   `json:"id"`
	ParentID  string   `json:"parent_id"`
	Resources []string `json:"resources"`
}

// UserSnapshot is a flat user record for serialization.
type UserSnapshot struct {
	ID     string `json:"id"`
	RoleID string `json:"role_id"`
}

// GrantSnapshot is a flat grant record for serialization.
type GrantSnapshot struct {
	FromRoleID string `json:"from_role_id"`
	ToRoleID   string `json:"to_role_id"`
	Resource   string `json:"resource"`
}
