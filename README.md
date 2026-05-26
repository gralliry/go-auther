# go-auther

[![Go Reference](https://pkg.go.dev/badge/github.com/gralliry/go-auther.svg)](https://pkg.go.dev/github.com/gralliry/go-auther)
[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/dl/)

Role-tree authorization library for Go. Explicit resource delegation with glob matching — no implicit inheritance.

## Installation

```sh
go get github.com/gralliry/go-auther
```

## Quick start

```go
package main

import (
    "fmt"
    "github.com/gralliry/go-auther"
    memory "github.com/gralliry/go-auther/adapters/memory"
)

func main() {
    a, _ := auther.NewAuthorizer(memory.New())

    // Build hierarchy: root -> admin -> editor
    _ = a.CreateRole("root", "admin")
    _ = a.CreateRole("admin", "editor")

    // Grant resources from root to roles
    _ = a.Grant("root", "admin", "/user/*")
    _ = a.Grant("root", "admin", "/reports/*")
    _ = a.Grant("root", "editor", "/data/*")

    // admin delegates /reports/* to editor
    _ = a.Grant("admin", "editor", "/reports/*")

    _ = a.CreateUser("editor", "alice")

    ok, _ := a.Enforce("alice", "/data/read")  // true
    ok, _ = a.Enforce("alice", "/reports/q1")  // true
    ok, _ = a.Enforce("alice", "/user/create") // false (not granted)
    fmt.Println(ok)
}
```

## Concepts

Permissions are **explicit-only**: a role only has access to resources granted to it by an ancestor. Root's `/**` does not leak to children.

```
root (/**)
 └── admin (/user/*, /reports/* from root)
       └── editor (/data/* from root, /reports/* from admin)
             └── user: alice

alice can:  /data/read, /reports/q1
alice cannot: /user/create (not granted), /** (no inheritance)
```

## API

### Roles

```go
func (a *Authorizer) CreateRole(parentID, roleID string) error
func (a *Authorizer) DeleteRole(roleID string) error
func (a *Authorizer) GetRole(roleID string) (*RoleInfo, error)
func (a *Authorizer) GetAllRoles() []*RoleInfo
func (a *Authorizer) GetSubRoles(roleID string) ([]*RoleInfo, error)
func (a *Authorizer) GetResource(roleID string) ([]string, error)
```

`DeleteRole` cascades: sub-roles and their users are removed, related grants cleaned up. Root cannot be deleted.

### Grants

```go
func (a *Authorizer) Grant(fromRoleID, toRoleID, resource string) error
func (a *Authorizer) Revoke(fromRoleID, toRoleID, resource string) error
func (a *Authorizer) GetGrantsTo(roleID string) ([]GrantNode, error)
func (a *Authorizer) GetGrantsFrom(roleID string) ([]GrantNode, error)
func (a *Authorizer) GetAllGrants() []GrantNode
```

`fromRoleID` must be an ancestor of `toRoleID`. Self-grant is not allowed. `Revoke` cascades: all descendant grants for the same resource are removed.

### Users

```go
func (a *Authorizer) CreateUser(roleID, userID string) error
func (a *Authorizer) DeleteUser(roleID, userID string) error
func (a *Authorizer) GetUser(userID string) (*UserInfo, error)
func (a *Authorizer) GetUsers() []*UserInfo
```

### Enforcement

```go
func (a *Authorizer) Enforce(userID, resource string) (bool, error)
func (a *Authorizer) Permissions(userID string) ([]string, error)
```

## Resource patterns

| Pattern | Matches |
|---|---|
| `/user/create` | Exact match only |
| `/user/*` | Single segment: `/user/123`, `/user/edit` |
| `/data/**` | Zero or more segments: `/data`, `/data/a/b/c` |
| `/**` | Everything |

Zero-allocation iterative backtracking matcher.

## Persistence

Write-through: every mutation is immediately persisted via the adapter.

```go
type Adapter interface {
    Load() (*snapshot.Policy, error)
    Save(snapshot *snapshot.Policy) error

    // Incremental methods use snapshot types for future-proofing.
    SetRole(role snapshot.Role) error
    UnsetRole(role snapshot.Role) error
    SetUser(user snapshot.User) error
    UnsetUser(user snapshot.User) error
    SetGrant(grant snapshot.Grant) error
    UnsetGrant(grant snapshot.Grant) error
}
```

**Memory** (testing / dev):

```go
import memory "github.com/gralliry/go-auther/adapters/memory"

a, _ := auther.NewAuthorizer(memory.New())
```

**JSON** (file-backed, atomic writes):

```go
import json "github.com/gralliry/go-auther/adapters/json"

a, _ := auther.NewAuthorizer(json.New("/path/to/policy.json"))
```

**SQL** (MySQL, PostgreSQL, SQLite — any `database/sql` driver):

```go
import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql" // or lib/pq, modernc.org/sqlite, etc.
    sql "github.com/gralliry/go-auther/adapters/sql"
)

db, _ := sql.Open("mysql", "user:pass@tcp(127.0.0.1:3306)/dbname")
adapter, _ := sql.New(db, "myapp_", "auther_policy")
a, _ := auther.NewAuthorizer(adapter)
```

Requires a non-nil adapter. Use `memory.New()` for in-memory-only mode.

### Custom adapter

Implement the `Adapter` interface to integrate any storage backend:

```go
type myAdapter struct {
    // your storage here
}

func (m *myAdapter) Load() (*snapshot.Policy, error)          { /* load from your store */ }
func (m *myAdapter) Save(s *snapshot.Policy) error            { /* save full snapshot */ }
func (m *myAdapter) SetRole(role snapshot.Role) error         { /* insert role */ }
func (m *myAdapter) UnsetRole(role snapshot.Role) error       { /* delete role */ }
func (m *myAdapter) SetUser(user snapshot.User) error         { /* insert user */ }
func (m *myAdapter) UnsetUser(user snapshot.User) error       { /* delete user */ }
func (m *myAdapter) SetGrant(grant snapshot.Grant) error      { /* insert grant */ }
func (m *myAdapter) UnsetGrant(grant snapshot.Grant) error    { /* delete grant */ }

a, _ := auther.NewAuthorizer(&myAdapter{})
```

For backends that can't do incremental writes (e.g., JSON file), implement the incremental methods by caching the snapshot and doing full rewrites — see `adapters/json` for a reference.

### Self-healing

On load, corrupted data is repaired: orphan roles reattached to root, dangling users/grants dropped, ancestor violations removed, duplicates deduplicated. Result is written back automatically.

## Performance

Enforcement hot path (i7-12700H):

| Scenario | Time |
|---|---|
| Exact match (GrantedMap O(1)) | ~38 ns |
| Wildcard match | ~40 ns |
| Grant-based hit | ~44 ns |
| Literal miss | ~39 ns |

Glob matching:

| Case | Time |
|---|---|
| Exact match | ~2 ns |
| Literal miss | ~5 ns |
| Single wildcard `*` | ~38 ns |
| Double wildcard `**` deep | ~66 ns |

## Errors

All sentinel errors work with `errors.Is`:

| Error | Meaning |
|---|---|
| `ErrAdapterRequired` | Adapter is nil |
| `ErrUserNotFound` | User does not exist |
| `ErrDuplicateUser` | User already exists |
| `ErrRoleNotFound` | Role does not exist |
| `ErrDuplicateRole` | Role already exists |
| `ErrGrantNotFound` | Grant (From, To, Resource) not found |
| `ErrDuplicateGrant` | Grant already exists |
| `ErrNotAncestor` | Grantor is not an ancestor of grantee |
| `ErrCircularRoleHierarchy` | Cycle detected in role tree |
| `ErrInvalidResource` | Resource path invalid |
| `ErrRootRoleDelete` | Cannot delete root role |

## License

MIT
