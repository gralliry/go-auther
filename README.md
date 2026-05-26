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
func (a *Authorizer) GetGrantsTo(roleID string) ([]*model.GrantNode, error)
func (a *Authorizer) GetGrantsFrom(roleID string) ([]*model.GrantNode, error)
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

**SQL** (MySQL, PostgreSQL, SQLite — any GORM-supported driver):

```go
import (
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    sqladapter "github.com/gralliry/go-auther/adapters/sql"
)

db, _ := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
a, _ := auther.NewAuthorizer(sqladapter.New(db))
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

All measurements on i7-12700H, 5-run avg, **0 B/op, 0 allocs/op**.

### Glob matching

| Case | Time/op |
|---|---|
| Exact match | ~4 ns |
| Literal miss | ~8 ns |
| Single wildcard `*` | ~44 ns |
| Double wildcard `**` | ~73 ns |
| Deep path `**` | ~102 ns |

### Enforcement (full path: role lookup + resource matching)

| Scenario | Time/op |
|---|---|
| Exact match (GrantedMap O(1)) | ~72 ns |
| Wildcard match | ~68 ns |
| Grant-based hit | ~76 ns |
| Literal miss (fast fail) | ~74 ns |
| Full miss (all patterns scanned) | ~77 ns |

### Permission modification

| Scenario | Time/op |
|---|---|
| Grant (shallow hierarchy) | ~334 ns |
| Grant to 10-level deep role | ~341 ns |
| Revoke | ~396 ns |
| Revoke with cascade (3-level chain) | ~1,235 ns |

### Concurrent read-write contention

Random probability-based distribution via per-goroutine RNG.

| Read/Write ratio | Time/op |
|---|---|
| 99% read + 1% write | ~683 ns |
| 90% read + 10% write | ~1,326 ns |
| 80% read + 20% write | ~1,553 ns |
| 70% read + 30% write | ~1,509 ns |
| 50% read + 50% write | ~1,917 ns |
| 100% write (pure write-lock contention) | ~2,047 ns |

## License

MIT
