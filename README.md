# go-auther

[![Go Reference](https://pkg.go.dev/badge/github.com/gralliry/go-auther.svg)](https://pkg.go.dev/github.com/gralliry/go-auther)
[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/dl/)

Role-based authorization library for Go with glob pattern resource matching. Permissions are **explicit-only** — a role only has access to resources that have been directly granted to it. No implicit inheritance between roles.

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
    "github.com/gralliry/go-auther/adapter/driver/noop"
)

func main() {
    m, _ := auther.NewManager(noop.New())

    m.CreateRole("admin")
    m.CreateRole("editor")

    // Grant resources from root to roles.
    m.Grant("root", "/user/*", "admin")
    m.Grant("root", "/reports/*", "admin")
    m.Grant("root", "/data/*", "editor")

    // Admin delegates /reports/* to editor.
    m.Grant("admin", "/reports/*", "editor")

    // Create a user and assign roles.
    m.CreateUser("alice")
    m.Assign("alice", "editor")

    ok, _ := m.EnforceByUser("alice", "/data/read")   // true
    ok, _ = m.EnforceByUser("alice", "/reports/q1")   // true
    ok, _ = m.EnforceByUser("alice", "/user/create")  // false
    fmt.Println(ok)
}
```

## Concepts

Permissions are **explicit-only**: a role only has access to resources explicitly granted to it via policies. No automatic inheritance.

```
root (/** built-in)
 ├── admin  ← granted /user/*, /reports/* from root
 │     └── editor  ← granted /data/* from root, /reports/* from admin
 │           └── user: alice

alice can:   /data/read, /reports/q1
alice cannot: /user/create, /**
```

Every grant creates a **Policy** that forms a DAG. Each policy tracks its `parents` counter (number of grantor policies that subsume it) and its `children`, enabling **cascade revocation**: revoking a policy also invalidates descendant policies that have no remaining valid parent.

Policy IDs are generated via [snowflake](https://github.com/bwmarrin/snowflake) — globally unique `int64` values.

## API

### Manager

```go
m, err := auther.NewManager(adapter)

// Role management
m.CreateRole("role-id")                          // → error
m.DeleteRole("role-id")                          // → error
m.Grant("grantor", "/path/*", "grantee")         // → error
m.Revoke("role-id", "/path/*")                   // → error
m.EnforceByRole("role-id", "/data/read")         // → (bool, error)

// User management
m.CreateUser("user-id")                          // → error
m.CheckUser("user-id")                           // → bool
m.DeleteUser("user-id")                          // → error
m.Assign("user-id", "role-id")                   // → error
m.Unassign("user-id", "role-id")                 // → error
m.IsAssigned("user-id", "role-id")               // → (bool, error)
m.EnforceByUser("user-id", "/data/read")         // → (bool, error)
```

### Resource patterns

Resource paths are string patterns used in `Grant`, `Revoke`, `EnforceByUser`, and `EnforceByRole`. They are normalized internally: duplicate `/` are collapsed and a leading `/` is always added.

| Pattern | Matches | Does NOT match |
|---|---|---|
| `/user/create` | `/user/create` only | `/user/123`, `/user/create/sub` |
| `/user/*` | `/user/123`, `/user/edit` | `/user`, `/user/123/sub` |
| `/data/**` | `/data`, `/data/a`, `/data/a/b/c` | `/user/data` |
| `/**` | Everything | — |
| `/api/*/logs/**` | `/api/v1/logs`, `/api/v2/logs/error/today` | `/api/logs`, `/other/v1/logs` |

`*` matches exactly one path segment. `**` matches zero or more segments (and ignores everything after it). Matching is zero-allocation — no regex, no string splitting.

The two wildcards can be combined: `*` restricts specific segments while `**` opens the tail.

## Persistence

Write-through: every mutation is persisted via the adapter before updating in-memory state.

```go
type Adapter interface {
    Snapshot() (Snapshot, error)

    CreateRole(role Role) error
    DeleteRole(role Role) error

    LinkUser(user User) error
    DeleteUser(user User) error
    UnlinkUser(user User) error

    CreatePolicy(policy Policy) error
    DeletePolicy(policyID int64) error
}
```

All entity types (`entity.Role`, `entity.User`, `entity.Policy`, `entity.Snapshot`) use plain Go primitives.

### Built-in adapters

**Noop** — no persistence. Good for development and testing.

```go
import "github.com/gralliry/go-auther/adapter/driver/noop"

m, _ := auther.NewManager(noop.New())
```

**JSON** — file-backed with atomic writes (write to `.tmp` then rename). Concurrency-safe.

```go
import "github.com/gralliry/go-auther/adapter/driver/json"

a, _ := json.New("/path/to/policy.json")
m, _ := auther.NewManager(a)
```

### Custom adapter

Implement the `adapter.Adapter` interface. Implementations must be concurrency-safe.

```go
import "github.com/gralliry/go-auther/entity"

type myAdapter struct { /* your storage */ }

func (a *myAdapter) Snapshot() (entity.Snapshot, error)        { /* ... */ }
func (a *myAdapter) CreateRole(role entity.Role) error            { /* ... */ }
func (a *myAdapter) DeleteRole(role entity.Role) error            { /* ... */ }
func (a *myAdapter) LinkUser(user entity.User) error              { /* ... */ }
func (a *myAdapter) DeleteUser(user entity.User) error            { /* ... */ }
func (a *myAdapter) UnlinkUser(user entity.User) error            { /* ... */ }
func (a *myAdapter) CreatePolicy(policy entity.Policy) error      { /* ... */ }
func (a *myAdapter) DeletePolicy(policyID int64) error             { /* ... */ }

m, _ := auther.NewManager(&myAdapter{})
```

## Performance

All measurements on i7-12700H (20 threads), Go 1.26.

### NewResource + Match

Pattern created each iteration, target as raw `string`.

| Case | Time/op | Alloc |
|---|---|---|
| Exact match | 81 ns | 152 B |
| Wildcard `*` match | 91 ns | 152 B |
| Double star `**` match | 78 ns | 152 B |

### Match — pre-created pattern

Pattern pre-created, target as raw `string`. Zero allocation.

| Case | Time/op | Alloc |
|---|---|---|
| Exact match | 13 ns | 0 B |
| Wildcard `*` match | 20 ns | 0 B |
| Double star `**` match | 13 ns | 0 B |
| No match | 14 ns | 0 B |
| Long path `**` match | 33 ns | 0 B |

### Match — unnormalized input

`Match` handles raw (untrusted) strings directly — no leading `/`, double slashes, trailing `/`.

| Input | Pattern | Time/op | Alloc |
|---|---|---|---|
| `user/create` | `/user/create` | 15 ns | 0 B |
| `//user//alice/edit` | `/user/*/edit` | 2 ns | 0 B |
| `/user/create/` | `/user/create` | 17 ns | 0 B |

### Enforcement

| Scenario | Time/op | Alloc |
|---|---|---|
| Exact match hit | 52 ns | 0 B |
| Wildcard match hit | 50 ns | 0 B |
| Literal miss | 50 ns | 0 B |
| Root enforce | 44 ns | 0 B |
| EnforceByUser | 100 ns | 0 B |
| 20 policies scanned | 198 ns | 0 B |

### Permission modification

| Scenario | Time/op | Alloc |
|---|---|---|
| Create role | 580 ns | 240 B |
| Delete role | 6440 ns | 2130 B |
| Create user | 865 ns | 150 B |
| Delete user | 370 ns | 63 B |
| Grant | 3150 ns | 1060 B |
| Revoke | 1400 ns | 740 B |
| Revoke cascade (3 levels) | 3890 ns | 2080 B |
| Assign user | 750 ns | 360 B |
| Unassign user | 310 ns | 0 B |

### Concurrency

4 goroutines sharing one `Manager`.

| Read/Write ratio | Time/op | Alloc |
|---|---|---|
| 99% read + 1% write | 3420 ns | 343 B |
| 90% read + 10% write | 4250 ns | 715 B |
| 70% read + 30% write | 6590 ns | 1680 B |
| 50% read + 50% write | 9350 ns | 2330 B |
| 100% write | 14930 ns | 4230 B |

## License

MIT
