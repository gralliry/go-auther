# CLAUDE.md

This file provides guidance to Claude Code when working in this repository.

## Build & Test

```sh
go build ./...          # main module
go test ./...           # main module tests
go vet ./...

# JSON adapter is an independent Go module
cd adapter/driver/msgpack && go build ./... && go test ./...

# Benchmarks
go test -bench . ./internal/resource/
go test -bench . .
```

No Makefile or task runner. Standard `go` toolchain only.

## Architecture

Go role-tree authorization library (`github.com/gralliry/go-auther`). Permissions are **explicit-only**: a role only has access to resources explicitly granted by an ancestor. No implicit inheritance.

### Package layout

```
auther.go                  # Public API — type aliases (Manager, Store, NewManager)
errors/                    # Sentinel errors (ErrRoleNotFound, ErrUserNotFound, etc.)
internal/
  manager/                 # Core: Manager, Role, Policy, user operations
    manager.go             #   Manager struct, New(), load(), generateID()
    role.go                #   Role struct, CreateRole, DeleteRole, Grant, Revoke, EnforceByRole
    policy.go              #   Policy struct, newPolicy, revoke, contains, within, match
    user.go                #   User ops: CreateUser, DeleteUser, Assign, Unassign, EnforceByUser
  resource/                # Path pattern normalization + glob matching
    resource.go            #   NewResource, Match, Contains — segment-based, zero-allocation
    resource_test.go       #   Unit tests
    resource_bench_test.go #   Benchmarks
  pkg/
    set/                   # Generic set types (see table below)
    algo/
      graph.go             #   PruneTree — DFS orphan removal (currently unused)
adapter/
  store.go                 #   Store interface (persistence contract)
  types.go                 #   Plain types: Role, User, Policy, Snapshot
  driver/
    empty/                 #   No-op adapter (dev/testing, single module)
    json/                  #   JSON file-backed adapter (independent Go module, atomic writes)
auther_bench_test.go       # Manager-level benchmarks
auther_test.go             # Integration tests
```

### Core domain model

**`Manager`** — orchestrator. Holds `roles map[string]*Role` and `users map[string]map[string]struct{}` (userID → set of roleIDs). Protected by `sync.RWMutex`. On construction loads snapshot from adapter, builds role/user/policy graph. Root role always present with policy `id=0` granting `/**` (`parents=1` so it's Valid). Policy IDs via `bwmarrin/snowflake`.

**`Role`** — has `srcGrants` (policies received from grantors) and `tarGrants` (policies granted to others). Both `*AutoCacheSet[*Policy]`.
- `enforce(res string) bool` — checks `srcGrants.Any` via `policy.match(res)`
- `grant(res, grantee, policyID)` — finds parent policies in srcGrants that `contains(res)`, finds child/sub policies in grantee, creates new Policy with correct `parents` counter and `children` DAG links. Does NOT call adapter — caller does.

**`User`** (in-memory) — holds assigned role IDs as `map[string]struct{}`. `EnforceByUser` iterates roles, returns true on first match. Silently cleans up stale role references during iteration.

**`Policy`** — represents a single resource grant. Forms a DAG: `parents int` (active parent counter), `children *AutoCacheSet[*Policy]`.
- `Valid() bool` → `p.parents > 0`
- `revoke(onDelete)` — sets `parents = -1`, decrements each child's parents counter; if a child reaches 0, recursively revokes. Calls `onDelete(id)` for each invalidated policy.
- `contains(*Resource) bool` / `within(*Resource) bool` / `match(string) bool` — delegate to `Resource` methods.

**`Resource`** — normalized path pattern stored as pre-split string segments (`Segs []string`). Wildcards: `*` matches one segment, `**` matches zero or more (terminates pattern — everything after `**` is discarded).
- `NewResource(raw string)` — parses raw path into segments, collapsing duplicate slashes, stripping content after `**`.
- `Match(target string) bool` — walks target byte-by-byte against pattern segments. Zero-allocation (no regex, no split).
- `Contains(*Resource) bool` — checks if pattern subsumes another segment-by-segment. `**` always returns true.
- `String() string` — rebuilds normalized path from segments via `strings.Join`.

### Persistence (`adapter/`)

**`Store` interface** — all methods take/pass entity types (not raw IDs):
```go
type Store interface {
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

Entity types (`adapter.Role`, `adapter.User`, `adapter.Policy`) use plain Go primitives. `User` stores one record per (ID, RoleID) pair, so a single user can have multiple records.

**`driver/noop/`** — no-op adapter. Snapshots are no-ops, Snapshot() returns empty snapshot. Single module (no nested go.mod).

**`driver/json/`** — JSON file-backed. Independent Go module (`adapter/driver/msgpack/go.mod`). Concurrency-safe via `sync.RWMutex`. Atomic writes (write to `.tmp` then rename). `LinkUser` checks duplicate by (ID, RoleID) combo, not just ID. `DeleteUser` removes all records for the given ID. `UnlinkUser` removes one specific (ID, RoleID) pair.

**Write-through pattern**: all Manager mutations persist via adapter first, then update in-memory state. The one exception is `grant()` which is purely in-memory — the caller (`Grant`) handles persistence before calling `grant()`.

| Manager operation | Adapter call(s) |
|---|---|
| `CreateRole` | `CreateRole` |
| `DeleteRole` | `DeleteRole` + `DeletePolicy` (cascade) |
| `CreateUser` | *(in-memory only, no adapter call)* |
| `DeleteUser` | `DeleteUser` |
| `Assign` | `LinkUser` |
| `Unassign` | `UnlinkUser` |
| `Grant` | `CreatePolicy` |
| `Revoke` | `DeletePolicy` + cascade |

### Internal packages (`internal/pkg/`)

**`set/`** — 5 generic set implementations:

| Type | Backing | Key access | Auto-clean |
|---|---|---|---|
| `AutoCacheSet[V]` | `map[V]struct{}` | value (Valid()) | Yes (traversal drops invalid) |
| `AutoCacheMap[K,V]` | `map[K]V` | `V.ID()` + Valid() | Yes (traversal drops invalid) |
| `CacheSet[V]` | `map[V]struct{}` | value (Valid()) | Explicit `GC()` |
| `ValueSet[K,V]` | `map[K]V` | `V.ID()` | No |
| `Set[K]` | `map[K]struct{}` | key | No |

All traversals (`Range`, `Any`, `All`, `Filter`, `Length`, `ToSlice`) in `AutoCacheSet/Map` silently drop entries where `Valid() == false`. Callers never see invalid data.

**`algo/`** — `PruneTree(rootID, parent map)` via DFS. Not currently used.

### Testing conventions

- Unit tests in `*_test.go` alongside source files
- Benchmarks in separate `*_bench_test.go` files
- Use `b.Loop()` (Go 1.24+) for benchmark loops — no explicit `b.ResetTimer()` needed
- Use `for range N` (Go 1.22+) for fixed-count loops
- Use `wg.Go(func() { ... })` (Go 1.22+) instead of `wg.Add(1)` + `go func() { defer wg.Done(); ... }()`

### Benchmarks

| File | Coverage |
|---|---|
| `internal/resource/resource_bench_test.go` | NewAndMatch, MatchString, Match variants |
| `auther_bench_test.go` | Enforcement, Grant, Revoke, Role/User CRUD, Concurrent read-write |

### Key conventions

- Sentinel errors in `errors/` use `errors.New` for `errors.Is` matching
- Manager fields are unexported; accessed through exported methods with read/write locks
- `Policy.parents` is an integer counter (not bool) — supports multi-parent DAG; valid when > 0
- `Policy.id == 0` reserved for root `/**` policy
- Public API surface is minimal: `auther.go` exposes `Manager`, `Store`, `NewManager`
- All source files have doc comments on types, functions, and methods — follow the existing pattern when adding new code
- Preference: use `codegraph_explore` MCP tool for code lookup before grep/Read
