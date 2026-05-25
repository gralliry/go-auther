// Package sqladapter provides a database/sql-backed adapter for Auther policy persistence.
//
// Works with any database/sql driver (MySQL, PostgreSQL, SQLite, etc.).
// The caller brings their own *sql.DB and driver.
//
// Usage:
//
//	import (
//	    "database/sql"
//	    _ "github.com/go-sql-driver/mysql"
//	    sqladapter "github.com/gralliry/auther/adapters/sql"
//	)
//
//	db, _ := sql.Open("mysql", "user:pass@tcp(127.0.0.1:3306)/dbname")
//	adapter, _ := sqladapter.NewSQLAdapter(db, "myapp_", "auther")
//	a, _ := auther.NewAuthorizer(adapter)
package sqladapter

import (
	"database/sql"
	"fmt"
	"regexp"

	"auther/snapshot"
)

var validTableRE = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// SQLAdapter persists policy data via database/sql using per-entity tables.
type SQLAdapter struct {
	db        *sql.DB
	rolesTbl  string
	usersTbl  string
	grantsTbl string
}


// NewSQLAdapter creates a new SQL-backed adapter.
//
// db must be an open *sql.DB connection. prefix is prepended to table to form
// the table name base, which is then used to create three tables:
// <prefix><table>_roles, <prefix><table>_users, <prefix><table>_grants.
// Pass "" for no prefix.
//
// Tables are automatically created if they don't exist.
func NewSQLAdapter(db *sql.DB, prefix, table string) (*SQLAdapter, error) {
	if prefix != "" && !validTableRE.MatchString(prefix) {
		return nil, fmt.Errorf("sqladapter: invalid prefix %q", prefix)
	}
	if !validTableRE.MatchString(table) {
		return nil, fmt.Errorf("sqladapter: invalid table name %q", table)
	}

	base := prefix + table
	a := &SQLAdapter{
		db:        db,
		rolesTbl:  base + "_roles",
		usersTbl:  base + "_users",
		grantsTbl: base + "_grants",
	}

	for _, ddl := range []string{
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id        TEXT PRIMARY KEY,
			parent_id TEXT NOT NULL
		)`, a.rolesTbl),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id      TEXT PRIMARY KEY,
			role_id TEXT NOT NULL
		)`, a.usersTbl),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			from_role_id TEXT NOT NULL,
			to_role_id   TEXT NOT NULL,
			resource     TEXT NOT NULL,
			PRIMARY KEY (from_role_id, to_role_id, resource)
		)`, a.grantsTbl),
	} {
		if _, err := db.Exec(ddl); err != nil {
			return nil, fmt.Errorf("sqladapter: create table: %w", err)
		}
	}

	return a, nil
}

// ---------------------------------------------------------------------------
// Full-snapshot interface (auther.Adapter)
// ---------------------------------------------------------------------------

// Load reads all rows and assembles a PolicySnapshot.
func (a *SQLAdapter) Load() (*snapshot.Policy, error) {
	snap := &snapshot.Policy{}

	rows, err := a.db.Query(fmt.Sprintf("SELECT id, parent_id FROM %s", a.rolesTbl))
	if err != nil {
		return nil, fmt.Errorf("sqladapter: load roles: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var rs snapshot.Role
		if err := rows.Scan(&rs.ID, &rs.ParentID); err != nil {
			return nil, fmt.Errorf("sqladapter: scan role: %w", err)
		}
		snap.Roles = append(snap.Roles, rs)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	userRows, err := a.db.Query(fmt.Sprintf("SELECT id, role_id FROM %s", a.usersTbl))
	if err != nil {
		return nil, fmt.Errorf("sqladapter: load users: %w", err)
	}
	defer userRows.Close()
	for userRows.Next() {
		var us snapshot.User
		if err := userRows.Scan(&us.ID, &us.RoleID); err != nil {
			return nil, fmt.Errorf("sqladapter: scan user: %w", err)
		}
		snap.Users = append(snap.Users, us)
	}
	if err := userRows.Err(); err != nil {
		return nil, err
	}

	grantRows, err := a.db.Query(fmt.Sprintf("SELECT from_role_id, to_role_id, resource FROM %s", a.grantsTbl))
	if err != nil {
		return nil, fmt.Errorf("sqladapter: load grants: %w", err)
	}
	defer grantRows.Close()
	for grantRows.Next() {
		var gs snapshot.Grant
		if err := grantRows.Scan(&gs.FromRoleID, &gs.ToRoleID, &gs.Resource); err != nil {
			return nil, fmt.Errorf("sqladapter: scan grant: %w", err)
		}
		snap.Grants = append(snap.Grants, gs)
	}
	if err := grantRows.Err(); err != nil {
		return nil, err
	}

	return snap, nil
}

// Save truncates all tables and re-inserts the full state in a transaction.
func (a *SQLAdapter) Save(snapshot *snapshot.Policy) error {
	tx, err := a.db.Begin()
	if err != nil {
		return fmt.Errorf("sqladapter: begin tx: %w", err)
	}
	defer tx.Rollback()

	for _, tbl := range []string{a.rolesTbl, a.usersTbl, a.grantsTbl} {
		if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s", tbl)); err != nil {
			return fmt.Errorf("sqladapter: truncate: %w", err)
		}
	}

	for _, rs := range snapshot.Roles {
		if _, err := tx.Exec(
			fmt.Sprintf("INSERT INTO %s (id, parent_id) VALUES (?, ?)", a.rolesTbl),
			rs.ID, rs.ParentID,
		); err != nil {
			return fmt.Errorf("sqladapter: insert role: %w", err)
		}
	}
	for _, us := range snapshot.Users {
		if _, err := tx.Exec(
			fmt.Sprintf("INSERT INTO %s (id, role_id) VALUES (?, ?)", a.usersTbl),
			us.ID, us.RoleID,
		); err != nil {
			return fmt.Errorf("sqladapter: insert user: %w", err)
		}
	}
	for _, gs := range snapshot.Grants {
		if _, err := tx.Exec(
			fmt.Sprintf("INSERT INTO %s (from_role_id, to_role_id, resource) VALUES (?, ?, ?)", a.grantsTbl),
			gs.FromRoleID, gs.ToRoleID, gs.Resource,
		); err != nil {
			return fmt.Errorf("sqladapter: insert grant: %w", err)
		}
	}

	return tx.Commit()
}

// ---------------------------------------------------------------------------
// Incremental interface (auther.Adapter)
// ---------------------------------------------------------------------------

// SetRole inserts a single role row.
func (a *SQLAdapter) SetRole(role snapshot.Role) error {
	_, err := a.db.Exec(
		fmt.Sprintf("INSERT INTO %s (id, parent_id) VALUES (?, ?)", a.rolesTbl),
		role.ID, role.ParentID,
	)
	if err != nil {
		return fmt.Errorf("sqladapter: create role: %w", err)
	}
	return nil
}

// UnsetRole removes a single role row.
func (a *SQLAdapter) UnsetRole(role snapshot.Role) error {
	_, err := a.db.Exec(
		fmt.Sprintf("DELETE FROM %s WHERE id = ?", a.rolesTbl),
		role.ID,
	)
	if err != nil {
		return fmt.Errorf("sqladapter: delete role: %w", err)
	}
	return nil
}

// SetUser inserts a single user row.
func (a *SQLAdapter) SetUser(user snapshot.User) error {
	_, err := a.db.Exec(
		fmt.Sprintf("INSERT INTO %s (id, role_id) VALUES (?, ?)", a.usersTbl),
		user.ID, user.RoleID,
	)
	if err != nil {
		return fmt.Errorf("sqladapter: create user: %w", err)
	}
	return nil
}

// UnsetUser removes a single user row.
func (a *SQLAdapter) UnsetUser(user snapshot.User) error {
	_, err := a.db.Exec(
		fmt.Sprintf("DELETE FROM %s WHERE id = ? AND role_id = ?", a.usersTbl),
		user.ID, user.RoleID,
	)
	if err != nil {
		return fmt.Errorf("sqladapter: delete user: %w", err)
	}
	return nil
}

// SetGrant inserts a single grant row.
func (a *SQLAdapter) SetGrant(grant snapshot.Grant) error {
	_, err := a.db.Exec(
		fmt.Sprintf("INSERT INTO %s (from_role_id, to_role_id, resource) VALUES (?, ?, ?)", a.grantsTbl),
		grant.FromRoleID, grant.ToRoleID, grant.Resource,
	)
	if err != nil {
		return fmt.Errorf("sqladapter: add grant: %w", err)
	}
	return nil
}

// UnsetGrant deletes a single grant row.
func (a *SQLAdapter) UnsetGrant(grant snapshot.Grant) error {
	_, err := a.db.Exec(
		fmt.Sprintf("DELETE FROM %s WHERE from_role_id = ? AND to_role_id = ? AND resource = ?", a.grantsTbl),
		grant.FromRoleID, grant.ToRoleID, grant.Resource,
	)
	if err != nil {
		return fmt.Errorf("sqladapter: remove grant: %w", err)
	}
	return nil
}
