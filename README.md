# Auther

[![Go Reference](https://pkg.go.dev/badge/auther.svg)](https://pkg.go.dev/auther)

Auther is a role-tree-based authorization library for Go. It models organization hierarchies with explicit resource delegation — no implicit inheritance, no surprises.

## Concepts

| Concept | Description |
|---|---|
| **Role** | A node in the role tree. The root role (`root`) is auto-created with the `/**` resource. Roles can create sub-roles and users. |
| **User** | A passive leaf created by a role. Users inherit only the permissions of their creating role and cannot manage resources or create other entities. |
| **Resource** | A path-like string (e.g. `/user/create`, `/data/**`) supporting glob matching with `*` (single segment) and `**` (zero or more segments). |
| **Grant** | An explicit resource delegation from one role to a descendant. Permissions do **not** auto-inherit — a parent must explicitly grant resources to sub-roles. |

### Explicit-only model

Promotions do **not** flow down the tree automatically. A child role only has access to:

1. Resources added directly to itself (via self-grant)
2. Resources explicitly granted to it by an ancestor

```
root (has /**)
 └── admin (has /user/* via self-grant)
       └── editor (has /data/* via self-grant, plus /user/* via grant from admin)
             └── user: alice

// alice can access /data/read (editor's own)
// alice can access /user/create (grant from admin)
// alice CANNOT access /** (root's resources — no auto-inheritance)
```

## Installation

```sh
go get auther
```

## Quick start

```go
package main

import (
    "fmt"
    "auther"
)

func main() {
    // Create an authorizer without persistence (in-memory only).
    a, _ := auther.NewAuthorizer(nil)

    // Build a role hierarchy.
    a.CreateRole("root", "admin")
    a.CreateRole("admin", "editor")

    // Give roles their own resources.
    a.GrantResource("admin", "admin", "/user/*")
    a.GrantResource("editor", "editor", "/data/*")

    // Delegate /reports/* from admin to editor.
    a.GrantResource("admin", "editor", "/reports/*")

    // Create users.
    a.CreateUser("editor", "alice")

    // Check permissions.
    ok, _ := a.Enforce("alice", "/data/read")   // true (own resource)
    ok, _ = a.Enforce("alice", "/reports/q1")   // true (granted by admin)
    ok, _ = a.Enforce("alice", "/user/create")  // false (not granted to editor)
}
```

## API

### Roles

```go
func (a *Authorizer) CreateRole(parentID, roleID string) error
func (a *Authorizer) DeleteRole(roleID string) error
func (a *Authorizer) GetRole(roleID string) (*RoleInfo, error)
func (a *Authorizer) GetAllRoles() []*RoleInfo
func (a *Authorizer) GetEffectiveRoleResources(roleID string) ([]string, error)
```

`DeleteRole` cascades: all sub-roles and their users are removed. Grants involving deleted roles are cleaned up. The root role cannot be deleted.

### Resources (via self-grant)

```go
// Add a resource to a role's own set.
a.GrantResource("admin", "admin", "/user/*")

// Remove it.
a.RevokeResource("admin", "admin", "/user/*")
```

### Grants

```go
func (a *Authorizer) GrantResource(fromRoleID, toRoleID, resource string) error
func (a *Authorizer) RevokeResource(fromRoleID, toRoleID, resource string) error
func (a *Authorizer) GetGrantsToRole(roleID string) ([]RoleGrant, error)
func (a *Authorizer) GetGrantsFromRole(roleID string) ([]RoleGrant, error)
func (a *Authorizer) GetAllGrants() []RoleGrant
```

`GrantResource` requires `fromRoleID` to be an ancestor (or self) of `toRoleID`. When the two IDs are equal, the resource is added directly to the role rather than creating a grant record.

`RevokeResource` cascades: grants for the same resource within the subtree are also removed.

### Users

```go
func (a *Authorizer) CreateUser(roleID, userID string) error
func (a *Authorizer) DeleteUser(userID string) error
func (a *Authorizer) GetUser(userID string) (*UserInfo, error)
func (a *Authorizer) GetAllUsers() []*UserInfo
```

### Enforcement

```go
func (a *Authorizer) Enforce(userID, resource string) (bool, error)
func (a *Authorizer) GetUserPermissions(userID string) ([]string, error)
```

`Enforce` checks a user's role resources and all grants received by that role. It does **not** walk up ancestor roles — only the user's direct role is checked.

### Errors

All sentinel errors are defined for use with `errors.Is`:

| Error | Meaning |
|---|---|
| `ErrUserNotFound` | User ID does not exist |
| `ErrDuplicateUser` | User ID already exists |
| `ErrRoleNotFound` | Role ID does not exist |
| `ErrDuplicateRole` | Role ID already exists |
| `ErrGrantNotFound` | Grant (From, To, Resource) not found |
| `ErrDuplicateGrant` | Grant already exists |
| `ErrNotAncestor` | Grantor is not an ancestor of the grantee |
| `ErrRootRoleDelete` | Attempted to delete the root role |

## Resource patterns

Resources follow filesystem-path semantics with two wildcards:

| Pattern | Matches |
|---|---|
| `/user/create` | Exact match only |
| `/user/*` | One segment: `/user/123`, `/user/edit` |
| `/data/**` | Zero or more segments: `/data`, `/data/a/b/c` |
| `/**` | Everything |

Matching uses a bottom-up DP algorithm. Segment parsing avoids string allocation for fast matching in hot enforcement paths.

## Persistence

Auther uses a **write-through** pattern: every mutation is immediately persisted via the adapter.

### Adapter interface

```go
type Adapter interface {
    Load() (*PolicySnapshot, error)
    Save(snapshot *PolicySnapshot) error
}
```

### Built-in adapters

**Memory** (for testing and development):

```go
import memoryadapter "auther/adapters/memory"

adapter := memoryadapter.NewMemoryAdapter()
a, _ := auther.NewAuthorizer(adapter)
```

**File** (JSON on disk, atomic writes via temp + rename):

```go
import fileadapter "auther/adapters/file"

adapter, _ := fileadapter.NewFileAdapter("/path/to/policy.json")
a, _ := auther.NewAuthorizer(adapter)
```

### No adapter

Pass `nil` to `NewAuthorizer` for an in-memory-only authorizer with no persistence:

```go
a, _ := auther.NewAuthorizer(nil)
```

### Self-healing

When loading from an adapter, `buildTree` validates and repairs corrupted data:

- Roles with dangling `ParentID` are reattached to root
- Users referencing non-existent roles are dropped
- Grants with missing From/To roles are dropped
- Grants violating the ancestor constraint are dropped
- Duplicate grants are deduplicated
- Self-grants in the grant list are merged into role resources

The cleansed snapshot is written back automatically.

## Performance

Benchmarks on the enforcement hot path (measured with `b.Loop()`):

| Scenario | Time |
|---|---|
| Exact match | ~135 ns |
| Wildcard hit | ~132 ns |
| Grant-based hit | ~262 ns |
| Literal miss (no wildcards) | ~148 ns |
| Full scan miss | ~158 ns |

Resource matching benchmarks:

| Case | Time |
|---|---|
| Exact match | ~1.4 ns |
| Literal miss | ~4 ns |
| Single wildcard | ~143 ns |
| Double wildcard deep | ~220 ns |

## Thread safety

`Authorizer` is safe for concurrent use. All public methods are protected by a `sync.RWMutex`. Read operations (Enforce, GetRole, etc.) acquire a read lock; write operations (CreateRole, GrantResource, etc.) acquire a write lock.

## License

MIT.
