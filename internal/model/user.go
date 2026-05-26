package model

// UserNode represents a user in the authorization system.
type UserNode struct {
	ID   string
	Role *RoleNode
}
