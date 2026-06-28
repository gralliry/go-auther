// Package sql provides a database/sql-backed adapter for Auther.
//
// The adapter accepts any *sql.DB, so it works with any SQL driver
// (SQLite, PostgreSQL, MySQL, etc.). Schema is auto-created on init.
package sql

import (
	"database/sql"
	"fmt"

	"github.com/gralliry/go-auther/adapter"
)

// Adapter persists Auther state to any SQL database via database/sql.
// It is safe for concurrent use — database/sql handles connection pooling and
// synchronization internally.
type Adapter struct {
	db *sql.DB
}

// New opens a SQL-backed adapter over the given database connection.
// Tables are created automatically if they do not already exist.
func New(db *sql.DB) (*Adapter, error) {
	a := &Adapter{db: db}
	if err := a.migrate(); err != nil {
		return nil, fmt.Errorf("sql adapter: migrate: %w", err)
	}
	return a, nil
}

// migrate creates the required tables if they are missing.
func (a *Adapter) migrate() error {
	_, err := a.db.Exec(`
		CREATE TABLE IF NOT EXISTS role (
			id TEXT NOT NULL PRIMARY KEY
		);

		CREATE TABLE IF NOT EXISTS user_assignment (
			user_id TEXT NOT NULL,
			role_id TEXT NOT NULL,
			PRIMARY KEY (user_id, role_id)
		);

		CREATE TABLE IF NOT EXISTS policy (
			id INTEGER NOT NULL PRIMARY KEY,
			grantor_role_id TEXT NOT NULL,
			grantee_role_id TEXT NOT NULL,
			resource TEXT NOT NULL
		);
	`)
	return err
}

// All returns a point-in-time snapshot of all persisted state.
func (a *Adapter) All() (adapter.Snapshot, error) {
	roles, err := a.allRoles()
	if err != nil {
		return adapter.Snapshot{}, err
	}
	users, err := a.allUsers()
	if err != nil {
		return adapter.Snapshot{}, err
	}
	policies, err := a.allPolicies()
	if err != nil {
		return adapter.Snapshot{}, err
	}
	return adapter.Snapshot{
		Role:   roles,
		User:   users,
		Policy: policies,
	}, nil
}

func (a *Adapter) allRoles() ([]adapter.Role, error) {
	rows, err := a.db.Query(`SELECT id FROM role`)
	if err != nil {
		return nil, fmt.Errorf("sql adapter: read roles: %w", err)
	}
	defer rows.Close()

	var out []adapter.Role
	for rows.Next() {
		var r adapter.Role
		if err := rows.Scan(&r.ID); err != nil {
			return nil, fmt.Errorf("sql adapter: scan role: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (a *Adapter) allUsers() ([]adapter.User, error) {
	rows, err := a.db.Query(`SELECT user_id, role_id FROM user_assignment`)
	if err != nil {
		return nil, fmt.Errorf("sql adapter: read users: %w", err)
	}
	defer rows.Close()

	var out []adapter.User
	for rows.Next() {
		var u adapter.User
		if err := rows.Scan(&u.ID, &u.RoleID); err != nil {
			return nil, fmt.Errorf("sql adapter: scan user: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (a *Adapter) allPolicies() ([]adapter.Policy, error) {
	rows, err := a.db.Query(`SELECT id, grantor_role_id, grantee_role_id, resource FROM policy`)
	if err != nil {
		return nil, fmt.Errorf("sql adapter: read policies: %w", err)
	}
	defer rows.Close()

	var out []adapter.Policy
	for rows.Next() {
		var p adapter.Policy
		if err := rows.Scan(&p.ID, &p.GrantorRoleID, &p.GranteeRoleID, &p.Resource); err != nil {
			return nil, fmt.Errorf("sql adapter: scan policy: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// CreateRole inserts a new role. Duplicate IDs are silently ignored.
func (a *Adapter) CreateRole(role adapter.Role) error {
	_, err := a.db.Exec(`INSERT OR IGNORE INTO role (id) VALUES (?)`, role.ID)
	if err != nil {
		return fmt.Errorf("sql adapter: create role %q: %w", role.ID, err)
	}
	return nil
}

// DeleteRole removes a role by ID. If the role does not exist, it is a no-op.
func (a *Adapter) DeleteRole(role adapter.Role) error {
	_, err := a.db.Exec(`DELETE FROM role WHERE id = ?`, role.ID)
	if err != nil {
		return fmt.Errorf("sql adapter: delete role %q: %w", role.ID, err)
	}
	return nil
}

// CreateUser adds a user-role assignment. Duplicate (ID, RoleID) pairs are
// silently ignored — the same user can hold multiple roles.
func (a *Adapter) CreateUser(user adapter.User) error {
	_, err := a.db.Exec(`INSERT OR IGNORE INTO user_assignment (user_id, role_id) VALUES (?, ?)`, user.ID, user.RoleID)
	if err != nil {
		return fmt.Errorf("sql adapter: create user %q: %w", user.ID, err)
	}
	return nil
}

// DeleteUser removes all role assignments for the given user ID.
func (a *Adapter) DeleteUser(user adapter.User) error {
	_, err := a.db.Exec(`DELETE FROM user_assignment WHERE user_id = ?`, user.ID)
	if err != nil {
		return fmt.Errorf("sql adapter: delete user %q: %w", user.ID, err)
	}
	return nil
}

// UnassignUser removes a single user-role assignment. If the (ID, RoleID) pair
// does not exist, it is a no-op.
func (a *Adapter) UnassignUser(user adapter.User) error {
	_, err := a.db.Exec(`DELETE FROM user_assignment WHERE user_id = ? AND role_id = ?`, user.ID, user.RoleID)
	if err != nil {
		return fmt.Errorf("sql adapter: unassign user %q role %q: %w", user.ID, user.RoleID, err)
	}
	return nil
}

// CreatePolicy inserts a new policy. Duplicate IDs are silently ignored.
func (a *Adapter) CreatePolicy(policy adapter.Policy) error {
	_, err := a.db.Exec(
		`INSERT OR IGNORE INTO policy (id, grantor_role_id, grantee_role_id, resource) VALUES (?, ?, ?, ?)`,
		policy.ID, policy.GrantorRoleID, policy.GranteeRoleID, policy.Resource,
	)
	if err != nil {
		return fmt.Errorf("sql adapter: create policy %d: %w", policy.ID, err)
	}
	return nil
}

// DeletePolicy removes a policy by ID. If the policy does not exist, it is a no-op.
func (a *Adapter) DeletePolicy(policyID int64) error {
	_, err := a.db.Exec(`DELETE FROM policy WHERE id = ?`, policyID)
	if err != nil {
		return fmt.Errorf("sql adapter: delete policy %d: %w", policyID, err)
	}
	return nil
}
