package model

// UserNode 表示权限系统中的一个用户。
type UserNode struct {
	ID   string
	Role *RoleNode
}
