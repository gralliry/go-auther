// Package sql provides a GORM-backed adapter for Auther policy persistence.
//
// Works with any GORM-supported database (MySQL, PostgreSQL, SQLite, etc.).
//
// Usage:
//
//	import (
//	    "gorm.io/driver/mysql"
//	    "gorm.io/gorm"
//	    sql "github.com/gralliry/go-auther/adapters/sql"
//	)
//
//	db, _ := gorm.Open(mysql.Open("user:pass@tcp(127.0.0.1:3306)/dbname"), &gorm.Config{})
//	adapter := sql.New(db)
//	a, _ := auther.NewAuthorizer(adapter)
package sql

import (
	"fmt"

	"github.com/gralliry/go-auther/adapters/sql/model"
	"github.com/gralliry/go-auther/snapshot"
	"gorm.io/gorm"
)

// Adapter persists policy data via GORM using per-entity tables.
type Adapter struct {
	db *gorm.DB
}

// New creates a new GORM-backed adapter.
// Tables are automatically created if they don't exist.
func New(db *gorm.DB) *Adapter {
	for _, m := range []any{&model.Role{}, &model.User{}, &model.Grant{}} {
		db.AutoMigrate(m)
	}
	return &Adapter{db: db}
}

// ---------------------------------------------------------------------------
// Full-snapshot interface (auther.Adapter)
// ---------------------------------------------------------------------------

// Load reads all rows and assembles a PolicySnapshot.
func (a *Adapter) Load() (*snapshot.Policy, error) {
	var roles []model.Role
	if err := a.db.Find(&roles).Error; err != nil {
		return nil, fmt.Errorf("sql: load roles: %w", err)
	}
	var users []model.User
	if err := a.db.Find(&users).Error; err != nil {
		return nil, fmt.Errorf("sql: load users: %w", err)
	}
	var grants []model.Grant
	if err := a.db.Find(&grants).Error; err != nil {
		return nil, fmt.Errorf("sql: load grants: %w", err)
	}

	return &snapshot.Policy{
		Roles:  model.Roles2Snapshots(roles),
		Users:  model.Users2Snapshots(users),
		Grants: model.Grants2Snapshots(grants),
	}, nil
}

// Save truncates all tables and re-inserts the full state in a transaction.
func (a *Adapter) Save(s *snapshot.Policy) error {
	return a.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("1 = 1").Delete(&model.Role{})
		tx.Where("1 = 1").Delete(&model.User{})
		tx.Where("1 = 1").Delete(&model.Grant{})

		if len(s.Roles) > 0 {
			if err := tx.Create(model.Roles2Models(s.Roles)).Error; err != nil {
				return fmt.Errorf("sql: insert roles: %w", err)
			}
		}
		if len(s.Users) > 0 {
			if err := tx.Create(model.Users2Models(s.Users)).Error; err != nil {
				return fmt.Errorf("sql: insert users: %w", err)
			}
		}
		if len(s.Grants) > 0 {
			if err := tx.Create(model.Grants2Models(s.Grants)).Error; err != nil {
				return fmt.Errorf("sql: insert grants: %w", err)
			}
		}

		return nil
	})
}

// ---------------------------------------------------------------------------
// Incremental interface (auther.Adapter)
// ---------------------------------------------------------------------------

func (a *Adapter) SetRole(role snapshot.Role) error {
	return a.db.Create(&model.Role{ID: role.ID, ParentID: role.ParentID}).Error
}

func (a *Adapter) UnsetRole(role snapshot.Role) error {
	return a.db.Delete(&model.Role{}, "id = ?", role.ID).Error
}

func (a *Adapter) SetUser(user snapshot.User) error {
	return a.db.Create(&model.User{ID: user.ID, RoleID: user.RoleID}).Error
}

func (a *Adapter) UnsetUser(user snapshot.User) error {
	return a.db.Delete(&model.User{}, "id = ? AND role_id = ?", user.ID, user.RoleID).Error
}

func (a *Adapter) SetGrant(grant snapshot.Grant) error {
	return a.db.Create(&model.Grant{FromRoleID: grant.FromRoleID, ToRoleID: grant.ToRoleID, Resource: grant.Resource}).Error
}

func (a *Adapter) UnsetGrant(grant snapshot.Grant) error {
	return a.db.Delete(&model.Grant{},
		"from_role_id = ? AND to_role_id = ? AND resource = ?",
		grant.FromRoleID, grant.ToRoleID, grant.Resource).Error
}
