# Auther

[![Go Reference](https://pkg.go.dev/badge/github.com/gralliry/auther.svg)](https://pkg.go.dev/github.com/gralliry/auther)

Role-tree authorization library for Go. Explicit resource delegation with glob matching — no implicit inheritance.

## Installation

```sh
go get github.com/gralliry/auther
```

## Quick start

```go
package main

import (
    "fmt"
    "github.com/gralliry/auther"
)

func main() {
    a, _ := auther.NewAuthorizer(nil)

    // Build hierarchy: root -> admin -> editor
    a.CreateRole("root", "admin")
    a.CreateRole("admin", "editor")

    // Self-grant resources to roles
    a.Grant("admin", "admin", "/user/*")
    a.Grant("editor", "editor", "/data/*")

    // Delegate /reports/* from admin to editor
    a.Grant("admin", "editor", "/reports/*")

    a.CreateUser("editor", "alice")

    ok, _ := a.Enforce("alice", "/data/read")  // true (own)
    ok, _ = a.Enforce("alice", "/reports/q1")  // true (granted)
    ok, _ = a.Enforce("alice", "/user/create") // false (not granted)
    fmt.Println(ok)
}
```

## Concepts

Permissions are **explicit-only**: a role only has access to resources directly granted to it via self-grant or delegated by an ancestor. Root's `/**` does not leak to children.

```
root (/**)
 └── admin (/user/*)
       └── editor (/data/*, /reports/* from admin)
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
func (a *Authorizer) Roles() []*RoleInfo
func (a *Authorizer) RoleResources(roleID string) ([]string, error)
```

`DeleteRole` cascades: sub-roles and their users are removed, related grants cleaned up. Root cannot be deleted.

### Grants

```go
func (a *Authorizer) Grant(fromRoleID, toRoleID, resource string) error
func (a *Authorizer) Revoke(fromRoleID, toRoleID, resource string) error
func (a *Authorizer) GrantsTo(roleID string) ([]GrantInfo, error)
func (a *Authorizer) GrantsFrom(roleID string) ([]GrantInfo, error)
func (a *Authorizer) AllGrants() []GrantInfo
```

When `fromRoleID == toRoleID`, the resource is added directly to the role (self-grant). Otherwise `fromRoleID` must be an ancestor of `toRoleID`. `Revoke` cascades: all descendant grants for the same resource are removed.

### Users

```go
func (a *Authorizer) CreateUser(roleID, userID string) error
func (a *Authorizer) DeleteUser(userID string) error
func (a *Authorizer) GetUser(userID string) (*UserInfo, error)
func (a *Authorizer) Users() []*UserInfo
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
    Load() (*PolicySnapshot, error)
    Save(snapshot *PolicySnapshot) error
}
```

**Memory** (testing / dev):

```go
import memoryadapter "github.com/gralliry/auther/adapters/memory"

a, _ := auther.NewAuthorizer(memoryadapter.NewMemoryAdapter())
```

**File** (JSON, atomic writes):

```go
import fileadapter "github.com/gralliry/auther/adapters/file"

a, _ := auther.NewAuthorizer(fileadapter.NewFileAdapter("/path/to/policy.json"))
```

**No adapter:** pass `nil` for in-memory-only.

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
