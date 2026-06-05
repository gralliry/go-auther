# go-auther

[![Go Reference](https://pkg.go.dev/badge/github.com/gralliry/go-auther.svg)](https://pkg.go.dev/github.com/gralliry/go-auther)
[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/dl/)

Role-based authorization library for Go with glob pattern resource matching. Permissions are explicit-only — no implicit inheritance.

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
    "github.com/gralliry/go-auther/adapter/empty"
)

func main() {
    m, _ := auther.New(empty.New())

    root, _ := m.GetRole("root")
    admin, _ := m.CreateRole("admin")
    editor, _ := m.CreateRole("editor")

    // Grant resources from root to roles.
    root.Grant(auther.Resource("/user/*"), admin)
    root.Grant(auther.Resource("/reports/*"), admin)
    root.Grant(auther.Resource("/data/*"), editor)

    // Admin delegates /reports/* to editor.
    admin.Grant(auther.Resource("/reports/*"), editor)

    // Create a user and assign a role.
    alice, _ := m.CreateUser("alice")
    alice.Assign(editor)

    ok, _ := alice.Enforce(auther.Resource("/data/read"))   // true
    ok, _ = alice.Enforce(auther.Resource("/reports/q1"))   // true
    ok, _ = alice.Enforce(auther.Resource("/user/create"))  // false
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

Every grant creates a **Policy** object that forms a DAG (directed acyclic graph). A policy tracks its `parents` and `children`, enabling cascade revocation: revoking a policy also revokes descendant policies that have no other valid parent.

Policy IDs are generated via [snowflake](https://github.com/bwmarrin/snowflake) — globally unique `int64` values.

## API

### Manager

The entry point. Created via `auther.New(adapter)`.

```go
m, _ := auther.New(adapter)

root, _  := m.GetRole("root")
role, _  := m.CreateRole("role-id")
user, _  := m.CreateUser("user-id")
user, _  := m.GetUser("user-id")
```

### Role

```go
role.ID() string
role.Valid() bool
role.Enforce(resource string) (bool, error)
role.Grant(resource *Resource, grantee *Role) (*Policy, error)
role.Revoke(policy *Policy) error
role.Delete() error
```

- `Grant` delegates a resource to another role. The grantor must already hold a policy that contains the resource. Self-grant is rejected.
- `Revoke` removes a policy and cascades to descendant policies that have no other valid parent.
- `Delete` invalidates the role and revokes all its policies. Persists the deletion to the adapter.

### User

```go
user.ID() string
user.Valid() bool
user.Assign(role *Role) error
user.Unassign(role *Role) error
user.IsAssign(role *Role) (bool, error)
user.Enforce(resource string) (bool, error)
user.Delete() error
```

A user holds multiple roles. `Enforce` succeeds if any assigned role has access.

### Policy

```go
policy.ID() int64
policy.Valid() bool
policy.Contains(resource *Resource) bool   // true if policy's resource pattern matches the target
policy.Within(resource *Resource) bool     // true if the target pattern contains this policy's resource
```

## Resource patterns

| Pattern | Matches |
|---|---|
| `/user/create` | Exact match only |
| `/user/*` | Single segment: `/user/123`, `/user/edit` |
| `/data/**` | Zero or more segments: `/data`, `/data/a/b/c` |
| `/**` | Everything |

The `*` wildcard matches exactly one path segment. The `**` wildcard matches zero or more segments. Matching uses zero-allocation byte-level scanning with early-exit fast paths.

The `Resource` type normalizes on construction: duplicate `/` are collapsed, a leading `/` is always present, and `**` is truncated to appear at most once at the last position.

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

All entity types (`adapter.Role`, `adapter.User`, `adapter.Policy`, `adapter.Snapshot`) use plain Go primitives — no dependency on `internal/model` types.

### Built-in adapters

**Empty** (in-memory, no persistence):

```go
import "github.com/gralliry/go-auther/adapter/empty"

m, _ := auther.New(empty.New())
```

**JSON** (file-backed, atomic writes):

```go
import "github.com/gralliry/go-auther/adapter/json"

m, _ := auther.New(json.New("/path/to/policy.json"))
```

### Custom adapter

Implement the `Adapter` interface:

```go
type myAdapter struct { /* your storage */ }

func (a *myAdapter) All() (adapter.Snapshot, error)          { /* ... */ }
func (a *myAdapter) CreateRole(role adapter.Role) error      { /* ... */ }
func (a *myAdapter) DeleteRole(roleID string) error           { /* ... */ }
func (a *myAdapter) CreateUser(user adapter.User) error       { /* ... */ }
func (a *myAdapter) DeleteUser(userID string) error           { /* ... */ }
func (a *myAdapter) CreatePolicy(policy adapter.Policy) error { /* ... */ }
func (a *myAdapter) DeletePolicy(policyID int64) error        { /* ... */ }

m, _ := auther.New(&myAdapter{})
```

Implementations must be concurrency-safe.

### Auto-cache pattern

The `internal/pkg/set` package provides `AutoCacheSet` and `AutoCacheMap` — generic collections that handle soft-deletion transparently. When an entity's `Valid()` returns false, it is lazily removed during the next traversal (`Range`, `Any`, `All`, `Filter`, etc.). Callers never need to check `Valid()` after reading from these collections.

## Performance

All measurements on i7-12700H (20 threads), Go 1.26. Enforcement and mutation benchmarks include lock overhead.

### Glob matching

| Case | Time/op | Alloc |
|---|---|---|
| Exact match | 26.7 ns | 0 B |
| Literal miss | 24.9 ns | 0 B |
| Single wildcard `*` | 20.4 ns | 0 B |
| Double wildcard `**` | 15.4 ns | 0 B |
| Deep path `**` | 22.2 ns | 0 B |

### Enforcement (role lookup + resource matching)

| Scenario | Time/op | Alloc |
|---|---|---|
| Exact match hit | 64.9 ns | 0 B |
| Wildcard match hit | 59.5 ns | 0 B |
| Literal miss (fast fail) | 57.3 ns | 0 B |
| Root enforce (`/**`) | 43.4 ns | 0 B |
| User enforce (user → role → policy) | 96.0 ns | 0 B |
| Enforce with 20 policies scanned | 226 ns | 0 B |

### Permission modification

| Scenario | Time/op | Alloc |
|---|---|---|
| Create role | 456 ns | 263 B |
| Grant | 1640 ns | 975 B |
| Revoke | 793 ns | 440 B |
| Revoke with 3-level cascade | 2523 ns | 1584 B |
| Assign user to role | 570 ns | 372 B |
| Delete role (with cascade) | 515 ns | 0 B |

### Concurrent read-write contention

4 goroutines sharing one `Area` (single `sync.RWMutex`).

| Read/Write ratio | Time/op | Alloc |
|---|---|---|
| 99% read + 1% write | 151 ns | 9 B |
| 90% read + 10% write | 524 ns | 92 B |
| 70% read + 30% write | 818 ns | 317 B |
| 50% read + 50% write | 1076 ns | 487 B |
| 100% write | 1813 ns | 871 B |

## Internal packages

| Package | Purpose |
|---|---|
| `internal/pkg/set` | Generic set types including auto-cache collections |
| `internal/pkg/algo` | `PruneTree` — DFS-based orphan node removal |
| `internal/pkg/strutil` | Key normalization with SHA-256 hashing for long keys |

## License

MIT
