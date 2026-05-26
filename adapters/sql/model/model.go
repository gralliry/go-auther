package model

var TableNamePrefix = "go_auther_"

// Role 是 roles 表的 GORM 模型。
type Role struct {
	ID       string `gorm:"primaryKey"`
	ParentID string `gorm:"not null"`
}

func (Role) TableName() string { return TableNamePrefix + "roles" }

// User 是 users 表的 GORM 模型。
type User struct {
	ID     string `gorm:"primaryKey"`
	RoleID string `gorm:"not null"`
}

func (User) TableName() string { return TableNamePrefix + "users" }

// Grant 是 grants 表的 GORM 模型。
type Grant struct {
	FromRoleID string `gorm:"primaryKey"`
	ToRoleID   string `gorm:"primaryKey"`
	Resource   string `gorm:"primaryKey"`
}

func (Grant) TableName() string { return TableNamePrefix + "grants" }
