# go-auther

[![Go Reference](https://pkg.go.dev/badge/github.com/gralliry/go-auther.svg)](https://pkg.go.dev/github.com/gralliry/go-auther)
[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/dl/)

Role-based authorization library for Go with glob pattern resource matching. Permissions are **explicit-only** — a role only has access to resources that have been directly granted to it through policies. No implicit inheritance between roles.

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
    "github.com/gralliry/go-auther/adapter/driver/empty"
)

func main() {
    m, _ := auther.NewManager(empty.New())

    m.CreateRole("admin")
    m.CreateRole("editor")

    // Grant resources from root to roles.
    m.Grant("root", "/user/*", "admin")
    m.Grant("root", "/reports/*", "admin")
    m.Grant("root", "/data/*", "editor")

    // Admin delegates /reports/* to editor.
    m.Grant("admin", "/reports/*", "editor")

    // Create a user and assign a role.
    m.CreateUser("alice")
    m.Assign("alice", "editor")

    ok, _ := m.EnforceByUser("alice", "/data/read")   // true
    ok, _ = m.EnforceByUser("alice", "/reports/q1")   // true
    ok, _ = m.EnforceByUser("alice", "/user/create")  // false
    fmt.Println(ok)
}
```

## Concepts

Permissions are **explicit-only**: a role only has access to resources explicitly granted to it via policies. There is no automatic inheritance between roles.

```
root (/** built-in)
 └── admin  ← granted /user/*, /reports/* from root
       └── editor  ← granted /data/* from root, /reports/* from admin
             └── user: alice

alice can:   /data/read, /reports/q1
alice cannot: /user/create, /**
```

Every grant creates a **Policy** object that forms a DAG (directed acyclic graph). A policy tracks its `parents` counter (the number of grantor policies that subsume it) and its `children`, enabling cascade revocation: revoking a policy also invalidates descendant policies that have no remaining valid parent.

Policy IDs are generated via [snowflake](https://github.com/bwmarrin/snowflake) — globally unique `int64` values.

## API

### Manager

The entry point. Created via `auther.NewManager(adapter)`.

```go
m, _ := auther.NewManager(adapter)

// Role management
m.CreateRole("role-id")                     // → error
role, ok := m.GetRole("role-id")            // → (*Role, bool)
m.DeleteRole("role-id")                     // → error
m.Grant("grantor", "/path/*", "grantee")    // → error
m.Revoke("role-id", "/path/*")              // → error
m.EnforceByRole("role-id", "/data/read")       // → (bool, error)

// User management
m.CreateUser("user-id")                     // → error
m.CheckUser("user-id")                      // → bool
m.DeleteUser("user-id")                     // → error
m.Assign("user-id", "role-id")              // → error
m.Unassign("user-id", "role-id")            // → error
m.IsAssigned("user-id", "role-id")          // → (bool, error)
m.EnforceByUser("user-id", "/data/read")       // → (bool, error)
```

All public mutation methods operate on role/user ID strings — `*Role` is only used as a read-only handle from `GetRole` for inspection.

```go
role, ok := m.GetRole("admin")
if ok {
    // role is a read-only handle; access checks go through the Manager
    ok, _ := m.EnforceByRole("admin", "/user/create")
}
```

## Resource patterns

Resource paths are string patterns used in `Grant`, `Revoke`, `EnforceByUser`, and `EnforceByRole` calls. They are normalized internally: duplicate `/` are collapsed and a leading `/` is always added.

| Pattern | Matches |
|---|---|
| `/user/create` | Exact match only |
| `/user/*` | Single segment: `/user/123`, `/user/edit` |
| `/data/**` | Zero or more segments: `/data`, `/data/a/b/c` |
| `/**` | Everything |

The `*` wildcard matches exactly one path segment. The `**` wildcard matches zero or more segments. Matching is zero-allocation.

## Persistence

Write-through: every mutation is immediately persisted via the adapter.

```go
type Adapter interface {
    All() (Snapshot, error)
    CreateRole(role Role) error
    DeleteRole(roleID string) error
    CreateUser(user User) error
    DeleteUser(userID string) error
    CreatePolicy(policy Policy) error
    DeletePolicy(policyID int64) error
}
```

All entity types (`adapter.Role`, `adapter.User`, `adapter.Policy`, `adapter.Snapshot`) use plain Go primitives.

### Built-in adapters

**Empty** (in-memory, no persistence):

```go
import "github.com/gralliry/go-auther/adapter/driver/empty"

m, _ := auther.NewManager(empty.New())
```

**JSON** (file-backed, atomic writes):

```go
import "github.com/gralliry/go-auther/adapter/driver/json"

m, _ := auther.NewManager(json.New("/path/to/policy.json"))
```

### Custom adapter

Implement the `adapter.Adapter` interface:

```go
type myAdapter struct { /* your storage */ }

func (a *myAdapter) All() (adapter.Snapshot, error)          { /* ... */ }
func (a *myAdapter) CreateRole(role adapter.Role) error      { /* ... */ }
func (a *myAdapter) DeleteRole(roleID string) error           { /* ... */ }
func (a *myAdapter) CreateUser(user adapter.User) error       { /* ... */ }
func (a *myAdapter) DeleteUser(userID string) error           { /* ... */ }
func (a *myAdapter) CreatePolicy(policy adapter.Policy) error { /* ... */ }
func (a *myAdapter) DeletePolicy(policyID int64) error        { /* ... */ }

m, _ := auther.NewManager(&myAdapter{})
```

Implementations must be concurrency-safe.

### Auto-cache pattern

Lazy cleanup for soft-deleted entities. When an entity is deleted, it is silently dropped during the next traversal. Callers never need to check validity after reading — they only ever see valid entries.

## Performance

All measurements on i7-12700H (20 threads), Go 1.26. `Match(string)` calls are zero-allocation; the "Resource creation + match" benchmarks below include the cost of constructing both pattern and target resources.

### Glob matching

| Case | Time/op | Alloc |
|---|---|---|
| Exact match | 29 ns | 0 B |
| Wildcard `*` match | 37 ns | 0 B |
| Double star `**` match | 33 ns | 0 B |

### Resource creation + match

| Case | Time/op | Alloc |
|---|---|---|
| Exact match | 197 ns | 88 B |
| Wildcard `*` match | 259 ns | 152 B |
| Double star `**` match | 298 ns | 280 B |
| Long path `**` match | 396 ns | 360 B |

### Enforcement

| Scenario | Time/op | Alloc |
|---|---|---|
| Exact match hit | 71 ns | 0 B |
| Wildcard match hit | 67 ns | 0 B |
| Literal miss | 63 ns | 0 B |
| Root enforce | 60 ns | 0 B |
| EnforceByUser | 105 ns | 0 B |
| 20 policies scanned | 255 ns | 0 B |

### Permission modification

| Scenario | Time/op | Alloc |
|---|---|---|
| Create role | 605 ns | 255 B |
| Grant | 1736 ns | 1083 B |
| Revoke | 1363 ns | 795 B |
| Delete role | 772 ns | 0 B |
| Assign user | 715 ns | 383 B |
| Revoke cascade (3 levels) | 17502 µs | 4055 KB |

### Concurrency

4 goroutines sharing one `Manager`.

| Read/Write ratio | Time/op | Alloc |
|---|---|---|
| 99% read + 1% write | 301 ns | 11 B |
| 90% read + 10% write | 1139 ns | 117 B |
| 70% read + 30% write | 1735 ns | 396 B |
| 50% read + 50% write | 2096 ns | 564 B |
| 100% write | 3668 ns | 1122 B |

## Internal packages

| Package | Purpose |
|---|---|
| `internal/manager` | Core authorization logic — `Manager`, `Role`, `Policy` |
| `internal/resource` | Path patterns with glob matching |
| `internal/pkg/set` | Generic set types |
| `internal/pkg/algo` | DFS-based orphan node removal |
| `errors` | Sentinel errors |

## License

MIT
