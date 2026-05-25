package model

// RoleGrant 表示从祖先角色到子角色的显式资源授权记录。
type RoleGrant struct {
	FromRoleID string
	ToRoleID   string
	Resource   string
}
