package sql

type roleModel struct {
	ID string `gorm:"primaryKey"`
}

func (roleModel) TableName() string { return "role" }

type userModel struct {
	UserID string `gorm:"primaryKey;column:user_id"`
	RoleID string `gorm:"primaryKey;column:role_id"`
}

func (userModel) TableName() string { return "user" }

type policyModel struct {
	ID            int64  `gorm:"primaryKey;autoIncrement:false"`
	GrantorRoleID string `gorm:"column:grantor_role_id"`
	GranteeRoleID string `gorm:"column:grantee_role_id"`
	Resource      string
}

func (policyModel) TableName() string { return "policy" }
