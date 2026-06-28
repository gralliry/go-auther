package auther

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gralliry/go-auther/adapter/driver/empty"
)

var benchBool bool

// =============================================================================
// Enforcement
// =============================================================================

func BenchmarkEnforceByRoleExactHit(b *testing.B) {
	m, _ := NewManager(empty.New())
	m.CreateRole("admin")
	m.Grant("root", "/user/create", "admin")

	for b.Loop() {
		benchBool, _ = m.EnforceByRole("admin", "/user/create")
	}
}

func BenchmarkEnforceByRoleWildcardHit(b *testing.B) {
	m, _ := NewManager(empty.New())
	m.CreateRole("admin")
	m.Grant("root", "/user/*", "admin")

	for b.Loop() {
		benchBool, _ = m.EnforceByRole("admin", "/user/create")
	}
}

func BenchmarkEnforceByRoleLiteralMiss(b *testing.B) {
	m, _ := NewManager(empty.New())
	m.CreateRole("admin")
	m.Grant("root", "/user/*", "admin")

	for b.Loop() {
		benchBool, _ = m.EnforceByRole("admin", "/data/read")
	}
}

func BenchmarkEnforceByRoleRoot(b *testing.B) {
	m, _ := NewManager(empty.New())

	for b.Loop() {
		benchBool, _ = m.EnforceByRole("root", "/anything")
	}
}

func BenchmarkEnforceByUser(b *testing.B) {
	m, _ := NewManager(empty.New())
	m.CreateRole("admin")
	m.Grant("root", "/user/*", "admin")
	m.CreateUser("alice")
	m.Assign("alice", "admin")

	for b.Loop() {
		benchBool, _ = m.EnforceByUser("alice", "/user/create")
	}
}

func BenchmarkEnforceByRoleManyPolicies(b *testing.B) {
	m, _ := NewManager(empty.New())
	m.CreateRole("admin")
	for i := range 20 {
		id := string(rune('a' + i))
		m.Grant("root", "/"+id+"/*", "admin")
	}

	for b.Loop() {
		benchBool, _ = m.EnforceByRole("admin", "/k/xyz")
	}
}

// =============================================================================
// Mutation — Role
// =============================================================================

func BenchmarkCreateRole(b *testing.B) {
	m, _ := NewManager(empty.New())
	var i int
	for b.Loop() {
		id := string(rune('A'+i%26)) + strconv.Itoa(i/26)
		i++
		m.CreateRole(id)
	}
}

func BenchmarkRoleDelete(b *testing.B) {
	m, _ := NewManager(empty.New())
	var i int
	for b.Loop() {
		roleID := "admin" + strconv.Itoa(i)
		editorID := "editor" + strconv.Itoa(i)
		i++
		m.CreateRole(roleID)
		m.CreateRole(editorID)
		m.Grant("root", "/user/*", roleID)
		m.Grant(roleID, "/user/profile", editorID)
		m.DeleteRole(roleID)
	}
}

// =============================================================================
// Mutation — Policy (Grant / Revoke)
// =============================================================================

func BenchmarkGrant(b *testing.B) {
	m, _ := NewManager(empty.New())
	var i int
	for b.Loop() {
		id := string(rune('A'+i%26)) + strconv.Itoa(i/26)
		i++
		m.CreateRole(id)
		m.Grant("root", "/data/*", id)
	}
}

func BenchmarkRevoke(b *testing.B) {
	m, _ := NewManager(empty.New())
	m.CreateRole("editor")
	m.Grant("root", "/data/*", "editor")

	for b.Loop() {
		m.Revoke("root", "/data/*")
		m.Grant("root", "/data/*", "editor")
	}
}

func BenchmarkRevokeCascade3Level(b *testing.B) {
	m, _ := NewManager(empty.New())
	m.CreateRole("a")
	m.CreateRole("b")
	m.CreateRole("c")

	for b.Loop() {
		m.Grant("root", "/data/**", "a")
		m.Grant("a", "/data/reports/*", "b")
		m.Grant("b", "/data/reports/q1", "c")
		m.Revoke("root", "/data/**")
	}
}

// =============================================================================
// Mutation — User (Assign / Unassign)
// =============================================================================

func BenchmarkUserAssign(b *testing.B) {
	m, _ := NewManager(empty.New())
	m.CreateRole("admin")
	var i int
	for b.Loop() {
		id := string(rune('A'+i%26)) + strconv.Itoa(i/26)
		i++
		m.CreateUser(id)
		m.Assign(id, "admin")
	}
}

func BenchmarkUserUnassign(b *testing.B) {
	m, _ := NewManager(empty.New())
	m.CreateRole("admin")
	m.CreateUser("alice")

	for b.Loop() {
		m.Assign("alice", "admin")
		m.Unassign("alice", "admin")
	}
}

func BenchmarkCreateUser(b *testing.B) {
	m, _ := NewManager(empty.New())
	var i int
	for b.Loop() {
		id := string(rune('A'+i%26)) + strconv.Itoa(i/26)
		i++
		m.CreateUser(id)
	}
}

func BenchmarkDeleteUser(b *testing.B) {
	m, _ := NewManager(empty.New())
	var i int
	for b.Loop() {
		id := string(rune('A'+i%26)) + strconv.Itoa(i/26)
		i++
		m.CreateUser(id)
		m.DeleteUser(id)
	}
}

// =============================================================================
// Concurrent read-write contention
// =============================================================================

func benchConcurrent(b *testing.B, readsPerWrite int) {
	m, _ := NewManager(empty.New())
	m.CreateRole("admin")
	m.Grant("root", "/user/*", "admin")
	m.CreateUser("alice")
	m.Assign("alice", "admin")

	var counter atomic.Int64
	var idCounter atomic.Int64

	for b.Loop() {
		var wg sync.WaitGroup
		for range 4 {
			wg.Go(func() {
				c := counter.Add(1)
				if c%int64(readsPerWrite) == 0 {
					id := idCounter.Add(1)
					name := string(rune('A'+id%26)) + strconv.Itoa(int(id/26))
					m.CreateRole(name)
					m.Grant("root", "/data/*", name)
				} else {
					m.EnforceByUser("alice", "/user/create")
				}
			})
		}
		wg.Wait()
	}
}

func BenchmarkConcurrent99Read1Write(b *testing.B)  { benchConcurrent(b, 100) }
func BenchmarkConcurrent90Read10Write(b *testing.B) { benchConcurrent(b, 10) }
func BenchmarkConcurrent70Read30Write(b *testing.B) { benchConcurrent(b, 3) }
func BenchmarkConcurrent50Read50Write(b *testing.B) { benchConcurrent(b, 2) }
func BenchmarkConcurrent100Write(b *testing.B)      { benchConcurrent(b, 1) }
