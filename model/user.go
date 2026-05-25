package model

// UserNode 表示权限系统中的一个用户。
type UserNode struct {
	ID   string
	Role *RoleNode
}

// UserInfo 是对外暴露的用户信息视图。
type UserInfo struct {
	ID     string
	RoleID string
}
