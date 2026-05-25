package auther

// PolicySnapshot 是完整权限状态的扁平化快照，由适配器用于持久化。
type PolicySnapshot struct {
	Roles  []RoleSnapshot  `json:"roles"`
	Users  []UserSnapshot  `json:"users"`
	Grants []GrantSnapshot `json:"grants"`
}

// RoleSnapshot 是用于序列化的扁平角色记录。
type RoleSnapshot struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id"`
}

// UserSnapshot 是用于序列化的扁平用户记录。
type UserSnapshot struct {
	ID     string `json:"id"`
	RoleID string `json:"role_id"`
}

// GrantSnapshot 是用于序列化的扁平授权记录。
type GrantSnapshot struct {
	FromRoleID string `json:"from_role_id"`
	ToRoleID   string `json:"to_role_id"`
	Resource   string `json:"resource"`
}
