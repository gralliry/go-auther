package model

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
