package adapter

type Role struct {
	ID string
}

type User struct {
	ID     string
	RoleID string
}

type Policy struct {
	ID       int64
	ParentID int64

	GrantorRoleID string
	GranteeRoleID string

	Resource string
}

type Snapshot struct {
	Role   []Role
	User   []User
	Policy []Policy
}
