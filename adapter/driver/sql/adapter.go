// Package sql provides a GORM-backed adapter for Auther.
//
// The adapter accepts any *gorm.DB, so it works with any SQL database
// supported by GORM (SQLite, PostgreSQL, MySQL, etc.). Schema is
// auto-migrated on init.
package sql

import (
	"fmt"

	"github.com/gralliry/go-auther/adapter"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Adapter persists Auther state to any SQL database via GORM.
type Adapter struct {
	db *gorm.DB
}

// New creates a SQL-backed adapter over the given GORM connection.
// Tables are auto-migrated if they do not already exist.
func New(db *gorm.DB) (*Adapter, error) {
	a := &Adapter{db: db}
	if err := a.migrate(); err != nil {
		return nil, fmt.Errorf("sql adapter: migrate: %w", err)
	}
	return a, nil
}

// migrate auto-creates the required tables via GORM's AutoMigrate.
func (a *Adapter) migrate() error {
	return a.db.AutoMigrate(&roleModel{}, &userModel{}, &policyModel{})
}

// ── Store implementation ─────────────────────────────────────────────────

// Snapshot returns a point-in-time copy of all persisted state.
func (a *Adapter) Snapshot() (adapter.Snapshot, error) {
	var roles []roleModel
	if err := a.db.Find(&roles).Error; err != nil {
		return adapter.Snapshot{}, fmt.Errorf("sql adapter: read roles: %w", err)
	}
	var users []userModel
	if err := a.db.Find(&users).Error; err != nil {
		return adapter.Snapshot{}, fmt.Errorf("sql adapter: read users: %w", err)
	}
	var policies []policyModel
	if err := a.db.Find(&policies).Error; err != nil {
		return adapter.Snapshot{}, fmt.Errorf("sql adapter: read policies: %w", err)
	}

	outRoles := make([]adapter.Role, len(roles))
	for i, r := range roles {
		outRoles[i] = adapter.Role{ID: r.ID}
	}
	outUsers := make([]adapter.User, len(users))
	for i, u := range users {
		outUsers[i] = adapter.User{ID: u.UserID, RoleID: u.RoleID}
	}
	outPolicies := make([]adapter.Policy, len(policies))
	for i, p := range policies {
		outPolicies[i] = adapter.Policy{
			ID:            p.ID,
			GrantorRoleID: p.GrantorRoleID,
			GranteeRoleID: p.GranteeRoleID,
			Resource:      p.Resource,
		}
	}

	return adapter.Snapshot{Role: outRoles, User: outUsers, Policy: outPolicies}, nil
}

// CreateRole inserts a new role. Duplicate IDs are silently ignored.
func (a *Adapter) CreateRole(role adapter.Role) error {
	return a.db.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&roleModel{ID: role.ID}).Error
}

// DeleteRole removes a role by ID. No-op if not found.
func (a *Adapter) DeleteRole(role adapter.Role) error {
	return a.db.Delete(&roleModel{ID: role.ID}).Error
}

// LinkUser adds a user-role binding. Duplicate pairs are silently ignored.
func (a *Adapter) LinkUser(user adapter.User) error {
	return a.db.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&userModel{UserID: user.ID, RoleID: user.RoleID}).Error
}

// DeleteUser removes all role bindings for the given user ID.
func (a *Adapter) DeleteUser(user adapter.User) error {
	return a.db.Where("user_id = ?", user.ID).
		Delete(&userModel{}).Error
}

// UnlinkUser removes a single user-role assignment. No-op if not found.
func (a *Adapter) UnlinkUser(user adapter.User) error {
	return a.db.Where("user_id = ? AND role_id = ?", user.ID, user.RoleID).
		Delete(&userModel{}).Error
}

// CreatePolicy inserts a new policy. Duplicate IDs are silently ignored.
func (a *Adapter) CreatePolicy(policy adapter.Policy) error {
	return a.db.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&policyModel{
			ID:            policy.ID,
			GrantorRoleID: policy.GrantorRoleID,
			GranteeRoleID: policy.GranteeRoleID,
			Resource:      policy.Resource,
		}).Error
}

// DeletePolicy removes a policy by ID. No-op if not found.
func (a *Adapter) DeletePolicy(policyID int64) error {
	return a.db.Delete(&policyModel{ID: policyID}).Error
}
