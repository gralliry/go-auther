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
//	adapter, _ := sqladapter.NewSQLAdapter(db, "auther_policy")
//	a, _ := auther.NewAuthorizer(adapter)
package sqladapter

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"

	"auther"
)

var validTableRE = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// TablePrefix is prepended to every table name passed to NewSQLAdapter.
// Set it once before creating adapters, e.g.:
//
//	sqladapter.TablePrefix = "myapp_"
//
// The default is empty (no prefix).
var TablePrefix string

// SQLAdapter persists policy snapshots via database/sql.
type SQLAdapter struct {
	db    *sql.DB
	table string
}

// NewSQLAdapter creates a new SQL-backed adapter.
//
// db must be an open *sql.DB connection. table is the table name used for
// policy storage — it is validated against /^[a-zA-Z_][a-zA-Z0-9_]*$/ to
// prevent SQL injection.
//
// The table is automatically created if it doesn't exist.
func NewSQLAdapter(db *sql.DB, table string) (*SQLAdapter, error) {
	if !validTableRE.MatchString(table) {
		return nil, fmt.Errorf("sqladapter: invalid table name %q", table)
	}

	fullTable := TablePrefix + table

	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		namespace TEXT PRIMARY KEY,
		data     TEXT NOT NULL
	)`, fullTable)

	if _, err := db.Exec(query); err != nil {
		return nil, fmt.Errorf("sqladapter: create table: %w", err)
	}

	return &SQLAdapter{db: db, table: fullTable}, nil
}

// Load reads the policy snapshot from the database.
// Returns nil if no snapshot has been saved yet.
func (a *SQLAdapter) Load() (*auther.PolicySnapshot, error) {
	query := fmt.Sprintf(`SELECT data FROM %s WHERE namespace = 'auther'`, a.table)
	var raw string
	err := a.db.QueryRow(query).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("sqladapter: load: %w", err)
	}

	var snap auther.PolicySnapshot
	if err := json.Unmarshal([]byte(raw), &snap); err != nil {
		return nil, fmt.Errorf("sqladapter: unmarshal: %w", err)
	}
	return &snap, nil
}

// Save persists the policy snapshot to the database.
// Uses DELETE + INSERT in a transaction for portable UPSERT behavior.
func (a *SQLAdapter) Save(snapshot *auther.PolicySnapshot) error {
	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("sqladapter: marshal: %w", err)
	}

	tx, err := a.db.Begin()
	if err != nil {
		return fmt.Errorf("sqladapter: begin tx: %w", err)
	}
	defer tx.Rollback()

	del := fmt.Sprintf(`DELETE FROM %s WHERE namespace = 'auther'`, a.table)
	if _, err := tx.Exec(del); err != nil {
		return fmt.Errorf("sqladapter: delete: %w", err)
	}

	ins := fmt.Sprintf(`INSERT INTO %s (namespace, data) VALUES ('auther', ?)`, a.table)
	if _, err := tx.Exec(ins, string(data)); err != nil {
		return fmt.Errorf("sqladapter: insert: %w", err)
	}

	return tx.Commit()
}
