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

// Clone 深拷贝一份 Policy，用于并发安全的快照分离。
func (p *Policy) Clone() *Policy {
	if p == nil {
		return nil
	}
	c := &Policy{}
	c.Roles = make([]Role, len(p.Roles))
	copy(c.Roles, p.Roles)
	c.Users = make([]User, len(p.Users))
	copy(c.Users, p.Users)
	c.Grants = make([]Grant, len(p.Grants))
	copy(c.Grants, p.Grants)
	return c
}
