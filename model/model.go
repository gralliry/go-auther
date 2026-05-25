// Package model 定义 auther 权限库的核心数据结构。
package model

// RoleNode 表示角色树中的一个角色节点。
type RoleNode struct {
	ID         string
	Parent     *RoleNode
	Children   map[string]*RoleNode
	Resources  map[string]bool
	GrantedMap map[string]bool // 索引：精确资源键 → true，O(1) 查找
	GrantsIn   []RoleGrant
	GrantsOut  []RoleGrant
	Users      map[string]*UserNode
}

// UserNode 表示权限系统中的一个用户。
type UserNode struct {
	ID   string
	Role *RoleNode
}

// RoleGrant 表示从祖先角色到子角色的显式资源授权记录。
type RoleGrant struct {
	FromRoleID string
	ToRoleID   string
	Resource   string
}

// RoleInfo 是对外暴露的角色信息视图。
type RoleInfo struct {
	ID         string
	ParentID   string
	Resources  []string
	SubRoleIDs []string
	UserIDs    []string
	GrantsIn   []RoleGrant
	GrantsOut  []RoleGrant
}

// UserInfo 是对外暴露的用户信息视图。
type UserInfo struct {
	ID     string
	RoleID string
}

// PolicySnapshot 是完整权限状态的扁平化快照，由适配器用于持久化。
type PolicySnapshot struct {
	Roles  []RoleSnapshot  `json:"roles"`
	Users  []UserSnapshot  `json:"users"`
	Grants []GrantSnapshot `json:"grants"`
}

// RoleSnapshot 是用于序列化的扁平角色记录。
type RoleSnapshot struct {
	ID        string   `json:"id"`
	ParentID  string   `json:"parent_id"`
	Resources []string `json:"resources"`
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
