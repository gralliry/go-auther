package auther

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gralliry/go-auther/adapter/driver/empty"
	"github.com/gralliry/go-auther/internal/resource"
)

var benchBool bool

// =============================================================================
// Resource create + match (combined cost)
// =============================================================================

func BenchmarkCreateMatchExact(b *testing.B) {
	for i := 0; i < b.N; i++ {
		r := resource.NewResource("/user/create")
		t := resource.NewResource("/user/create")
		benchBool = r.Match(t.String())
	}
}

func BenchmarkCreateMatchWildcard(b *testing.B) {
	for i := 0; i < b.N; i++ {
		r := resource.NewResource("/user/*/edit")
		t := resource.NewResource("/user/alice/edit")
		benchBool = r.Match(t.String())
	}
}

func BenchmarkCreateMatchDoubleStar(b *testing.B) {
	for i := 0; i < b.N; i++ {
		r := resource.NewResource("/api/v1/**")
		t := resource.NewResource("/api/v1/users/admin/data")
		benchBool = r.Match(t.String())
	}
}

// =============================================================================
// Match via Resource.Match(string) — string path (avoids resource.NewResource allocation)
// =============================================================================

func BenchmarkMatchStringExact(b *testing.B) {
	r := resource.NewResource("/user/create")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool = r.Match("/user/create")
	}
}

func BenchmarkMatchStringWildcard(b *testing.B) {
	r := resource.NewResource("/user/*/edit")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool = r.Match("/user/alice/edit")
	}
}

func BenchmarkMatchStringDoubleStar(b *testing.B) {
	r := resource.NewResource("/api/v1/**")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool = r.Match("/api/v1/users/admin/data")
	}
}

// =============================================================================
// Enforcement
// =============================================================================

func BenchmarkEnforceByRoleExactHit(b *testing.B) {
	m, _ := NewManager(empty.New())
	m.CreateRole("admin")
	m.Grant("root", "/user/create", "admin")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool, _ = m.EnforceByRole("admin", "/user/create")
	}
}

func BenchmarkEnforceByRoleWildcardHit(b *testing.B) {
	m, _ := NewManager(empty.New())
	m.CreateRole("admin")
	m.Grant("root", "/user/*", "admin")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool, _ = m.EnforceByRole("admin", "/user/create")
	}
}

func BenchmarkEnforceByRoleLiteralMiss(b *testing.B) {
	m, _ := NewManager(empty.New())
	m.CreateRole("admin")
	m.Grant("root", "/user/*", "admin")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool, _ = m.EnforceByRole("admin", "/data/read")
	}
}

func BenchmarkEnforceByRoleRoot(b *testing.B) {
	m, _ := NewManager(empty.New())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool, _ = m.EnforceByRole("root", "/anything")
	}
}

func BenchmarkEnforceByUser(b *testing.B) {
	m, _ := NewManager(empty.New())
	m.CreateRole("admin")
	m.Grant("root", "/user/*", "admin")
	m.CreateUser("alice")
	m.Assign("alice", "admin")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool, _ = m.EnforceByUser("alice", "/user/create")
	}
}

func BenchmarkEnforceByRoleManyPolicies(b *testing.B) {
	m, _ := NewManager(empty.New())
	m.CreateRole("admin")
	for i := 0; i < 20; i++ {
		id := string(rune('a' + i))
		m.Grant("root", "/"+id+"/*", "admin")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool, _ = m.EnforceByRole("admin", "/k/xyz")
	}
}

// =============================================================================
// Mutation
// =============================================================================

func BenchmarkGrant(b *testing.B) {
	m, _ := NewManager(empty.New())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := string(rune('A'+i%26)) + strconv.Itoa(i/26)
		m.CreateRole(id)
		m.Grant("root", "/data/*", id)
	}
}

func BenchmarkRevoke(b *testing.B) {
	m, _ := NewManager(empty.New())
	m.CreateRole("editor")
	m.Grant("root", "/data/*", "editor")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Revoke("root", "/data/*")
		m.Grant("root", "/data/*", "editor")
	}
}

func BenchmarkRevokeCascade3Level(b *testing.B) {
	m, _ := NewManager(empty.New())
	m.CreateRole("a")
	m.CreateRole("b")
	m.CreateRole("c")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Grant("root", "/data/**", "a")
		m.Grant("a", "/data/reports/*", "b")
		m.Grant("b", "/data/reports/q1", "c")
		m.Revoke("root", "/user/*")
	}
}

func BenchmarkRoleDelete(b *testing.B) {
	m, _ := NewManager(empty.New())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		roleID := "admin" + strconv.Itoa(i)
		editorID := "editor" + strconv.Itoa(i)
		m.CreateRole(roleID)
		m.CreateRole(editorID)
		m.Grant("root", "/user/*", roleID)
		m.Grant(roleID, "/user/profile", editorID)
		b.StartTimer()

		m.DeleteRole(roleID)
	}
}

func BenchmarkUserAssign(b *testing.B) {
	m, _ := NewManager(empty.New())
	m.CreateRole("admin")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := string(rune('A'+i%26)) + strconv.Itoa(i/26)
		m.CreateUser(id)
		m.Assign(id, "admin")
	}
}

func BenchmarkCreateRole(b *testing.B) {
	m, _ := NewManager(empty.New())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := string(rune('A'+i%26)) + strconv.Itoa(i/26)
		m.CreateRole(id)
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

	var roleCounter atomic.Int64
	b.ResetTimer()

	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n := b.N / 4
			for i := 0; i < n; i++ {
				if i%readsPerWrite == 0 {
					id := roleCounter.Add(1)
					m.CreateRole(string(rune('A'+id%26)) + strconv.Itoa(int(id/26)))
					m.Grant("root", "/data/*", string(rune('A'+id%26))+strconv.Itoa(int(id/26)))
				} else {
					m.EnforceByUser("alice", "/user/create")
				}
			}
		}()
	}
	wg.Wait()
}

func BenchmarkConcurrent99Read1Write(b *testing.B)  { benchConcurrent(b, 100) }
func BenchmarkConcurrent90Read10Write(b *testing.B) { benchConcurrent(b, 10) }
func BenchmarkConcurrent70Read30Write(b *testing.B) { benchConcurrent(b, 3) }
func BenchmarkConcurrent50Read50Write(b *testing.B) { benchConcurrent(b, 2) }
func BenchmarkConcurrent100Write(b *testing.B)      { benchConcurrent(b, 1) }
