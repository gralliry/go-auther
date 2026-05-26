// Package sql provides a database/sql-backed adapter for Auther policy persistence.
//
// Works with any database/sql driver (MySQL, PostgreSQL, SQLite, etc.).
// The caller brings their own *sql.DB and driver.
//
// Usage:
//
//	import (
//	    "database/sql"
//	    _ "github.com/go-sql-driver/mysql"
//	    sql "github.com/gralliry/go-auther/adapters/sql"
//	)
//
//	db, _ := sql.Open("mysql", "user:pass@tcp(127.0.0.1:3306)/dbname")
//	adapter, _ := sql.New(db, "myapp_", "auther")
//	a, _ := auther.NewAuthorizer(adapter)
package sql

import (
	"database/sql"
	"fmt"
	"regexp"

	"github.com/gralliry/go-auther/snapshot"
)

var validTableRE = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Adapter persists policy data via database/sql using per-entity tables.
type Adapter struct {
	db        *sql.DB
	rolesTbl  string
	usersTbl  string
	grantsTbl string
}

// New creates a new SQL-backed adapter.
//
// db must be an open *sql.DB connection. prefix is prepended to table to form
// the table name base, which is then used to create three tables:
// <prefix><table>_roles, <prefix><table>_users, <prefix><table>_grants.
// Pass "" for no prefix.
//
// Tables are automatically created if they don't exist.
func New(db *sql.DB, prefix, table string) (*Adapter, error) {
	if prefix != "" && !validTableRE.MatchString(prefix) {
		return nil, fmt.Errorf("sql: invalid prefix %q", prefix)
	}
	if !validTableRE.MatchString(table) {
		return nil, fmt.Errorf("sql: invalid table name %q", table)
	}

	base := prefix + table
	a := &Adapter{
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
			return nil, fmt.Errorf("sql: create table: %w", err)
		}
	}

	return a, nil
}

// ---------------------------------------------------------------------------
// Full-snapshot interface (auther.Adapter)
// ---------------------------------------------------------------------------

// Load reads all rows and assembles a PolicySnapshot.
func (a *Adapter) Load() (*snapshot.Policy, error) {
	snap := &snapshot.Policy{}

	rows, err := a.db.Query(fmt.Sprintf("SELECT id, parent_id FROM %s", a.rolesTbl))
	if err != nil {
		return nil, fmt.Errorf("sql: load roles: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var rs snapshot.Role
		if err := rows.Scan(&rs.ID, &rs.ParentID); err != nil {
			return nil, fmt.Errorf("sql: scan role: %w", err)
		}
		snap.Roles = append(snap.Roles, rs)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	userRows, err := a.db.Query(fmt.Sprintf("SELECT id, role_id FROM %s", a.usersTbl))
	if err != nil {
		return nil, fmt.Errorf("sql: load users: %w", err)
	}
	defer userRows.Close()
	for userRows.Next() {
		var us snapshot.User
		if err := userRows.Scan(&us.ID, &us.RoleID); err != nil {
			return nil, fmt.Errorf("sql: scan user: %w", err)
		}
		snap.Users = append(snap.Users, us)
	}
	if err := userRows.Err(); err != nil {
		return nil, err
	}

	grantRows, err := a.db.Query(fmt.Sprintf("SELECT from_role_id, to_role_id, resource FROM %s", a.grantsTbl))
	if err != nil {
		return nil, fmt.Errorf("sql: load grants: %w", err)
	}
	defer grantRows.Close()
	for grantRows.Next() {
		var gs snapshot.Grant
		if err := grantRows.Scan(&gs.FromRoleID, &gs.ToRoleID, &gs.Resource); err != nil {
			return nil, fmt.Errorf("sql: scan grant: %w", err)
		}
		snap.Grants = append(snap.Grants, gs)
	}
	if err := grantRows.Err(); err != nil {
		return nil, err
	}

	return snap, nil
}

// Save truncates all tables and re-inserts the full state in a transaction.
func (a *Adapter) Save(snapshot *snapshot.Policy) error {
	tx, err := a.db.Begin()
	if err != nil {
		return fmt.Errorf("sql: begin tx: %w", err)
	}
	defer tx.Rollback()

	for _, tbl := range []string{a.rolesTbl, a.usersTbl, a.grantsTbl} {
		if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s", tbl)); err != nil {
			return fmt.Errorf("sql: truncate: %w", err)
		}
	}

	for _, rs := range snapshot.Roles {
		if _, err := tx.Exec(
			fmt.Sprintf("INSERT INTO %s (id, parent_id) VALUES (?, ?)", a.rolesTbl),
			rs.ID, rs.ParentID,
		); err != nil {
			return fmt.Errorf("sql: insert role: %w", err)
		}
	}
	for _, us := range snapshot.Users {
		if _, err := tx.Exec(
			fmt.Sprintf("INSERT INTO %s (id, role_id) VALUES (?, ?)", a.usersTbl),
			us.ID, us.RoleID,
		); err != nil {
			return fmt.Errorf("sql: insert user: %w", err)
		}
	}
	for _, gs := range snapshot.Grants {
		if _, err := tx.Exec(
			fmt.Sprintf("INSERT INTO %s (from_role_id, to_role_id, resource) VALUES (?, ?, ?)", a.grantsTbl),
			gs.FromRoleID, gs.ToRoleID, gs.Resource,
		); err != nil {
			return fmt.Errorf("sql: insert grant: %w", err)
		}
	}

	return tx.Commit()
}

// ---------------------------------------------------------------------------
// Incremental interface (auther.Adapter)
// ---------------------------------------------------------------------------

func (a *Adapter) SetRole(role snapshot.Role) error {
	_, err := a.db.Exec(
		fmt.Sprintf("INSERT INTO %s (id, parent_id) VALUES (?, ?)", a.rolesTbl),
		role.ID, role.ParentID,
	)
	if err != nil {
		return fmt.Errorf("sql: create role: %w", err)
	}
	return nil
}

func (a *Adapter) UnsetRole(role snapshot.Role) error {
	_, err := a.db.Exec(
		fmt.Sprintf("DELETE FROM %s WHERE id = ?", a.rolesTbl),
		role.ID,
	)
	if err != nil {
		return fmt.Errorf("sql: delete role: %w", err)
	}
	return nil
}

func (a *Adapter) SetUser(user snapshot.User) error {
	_, err := a.db.Exec(
		fmt.Sprintf("INSERT INTO %s (id, role_id) VALUES (?, ?)", a.usersTbl),
		user.ID, user.RoleID,
	)
	if err != nil {
		return fmt.Errorf("sql: create user: %w", err)
	}
	return nil
}

func (a *Adapter) UnsetUser(user snapshot.User) error {
	_, err := a.db.Exec(
		fmt.Sprintf("DELETE FROM %s WHERE id = ? AND role_id = ?", a.usersTbl),
		user.ID, user.RoleID,
	)
	if err != nil {
		return fmt.Errorf("sql: delete user: %w", err)
	}
	return nil
}

func (a *Adapter) SetGrant(grant snapshot.Grant) error {
	_, err := a.db.Exec(
		fmt.Sprintf("INSERT INTO %s (from_role_id, to_role_id, resource) VALUES (?, ?, ?)", a.grantsTbl),
		grant.FromRoleID, grant.ToRoleID, grant.Resource,
	)
	if err != nil {
		return fmt.Errorf("sql: add grant: %w", err)
	}
	return nil
}

func (a *Adapter) UnsetGrant(grant snapshot.Grant) error {
	_, err := a.db.Exec(
		fmt.Sprintf("DELETE FROM %s WHERE from_role_id = ? AND to_role_id = ? AND resource = ?", a.grantsTbl),
		grant.FromRoleID, grant.ToRoleID, grant.Resource,
	)
	if err != nil {
		return fmt.Errorf("sql: remove grant: %w", err)
	}
	return nil
}
