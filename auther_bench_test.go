package auther

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gralliry/go-auther/adapter/empty"
)

// ---------------------------------------------------------------------------
// Glob matching
// ---------------------------------------------------------------------------

var benchBool bool

func BenchmarkMatchExact(b *testing.B) {
	r := NewResource("/user/create")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool = r.Match("/user/create")
	}
}

func BenchmarkMatchLiteralMiss(b *testing.B) {
	r := NewResource("/user/create")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool = r.Match("/user/delete")
	}
}

func BenchmarkMatchSingleStar(b *testing.B) {
	r := NewResource("/user/*")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool = r.Match("/user/create")
	}
}

func BenchmarkMatchDoubleStar(b *testing.B) {
	r := NewResource("/data/**")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool = r.Match("/data/a/b/c")
	}
}

func BenchmarkMatchDeepDoubleStar(b *testing.B) {
	r := NewResource("/api/v1/**")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool = r.Match("/api/v1/users/admin/permissions/read/write")
	}
}

// ---------------------------------------------------------------------------
// Enforcement
// ---------------------------------------------------------------------------

func BenchmarkEnforceExactHit(b *testing.B) {
	m, _ := New(empty.New())
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")
	root.Grant(NewResource("/user/create"), admin)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool, _ = admin.Enforce(NewResource("/user/create"))
	}
}

func BenchmarkEnforceWildcardHit(b *testing.B) {
	m, _ := New(empty.New())
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")
	root.Grant(NewResource("/user/*"), admin)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool, _ = admin.Enforce(NewResource("/user/create"))
	}
}

func BenchmarkEnforceLiteralMiss(b *testing.B) {
	m, _ := New(empty.New())
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")
	root.Grant(NewResource("/user/*"), admin)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool, _ = admin.Enforce(NewResource("/data/read"))
	}
}

// ---------------------------------------------------------------------------
// Mutation
// ---------------------------------------------------------------------------

func BenchmarkGrant(b *testing.B) {
	m, _ := New(empty.New())
	root, _ := m.GetRole("root")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := string(rune('A'+i%26)) + string(rune('0'+i/26))
		editor, _ := m.CreateRole(id)
		root.Grant(NewResource("/data/*"), editor)
	}
}

func BenchmarkRevoke(b *testing.B) {
	m, _ := New(empty.New())
	root, _ := m.GetRole("root")
	editor, _ := m.CreateRole("editor")
	policy, _ := root.Grant(NewResource("/data/*"), editor)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root.Revoke(policy)
		policy, _ = root.Grant(NewResource("/data/*"), editor)
	}
}

func BenchmarkRevokeCascade3Level(b *testing.B) {
	m, _ := New(empty.New())
	root, _ := m.GetRole("root")
	roleA, _ := m.CreateRole("a")
	roleB, _ := m.CreateRole("b")
	roleC, _ := m.CreateRole("c")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p, _ := root.Grant(NewResource("/data/**"), roleA)
		roleA.Grant(NewResource("/data/reports/*"), roleB)
		roleB.Grant(NewResource("/data/reports/q1"), roleC)
		root.Revoke(p)
	}
}

// ---------------------------------------------------------------------------
// Concurrent read-write contention
// ---------------------------------------------------------------------------

func benchConcurrent(b *testing.B, readsPerWrite int) {
	m, _ := New(empty.New())
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")
	root.Grant(NewResource("/user/*"), admin)
	alice, _ := m.CreateUser("alice")
	alice.Assign(admin)

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
					r, _ := m.CreateRole(string(rune('A'+id%26)) + string(rune('0'+id/26)))
					root.Grant(NewResource("/data/*"), r)
				} else {
					alice.Enforce(NewResource("/user/create"))
				}
			}
		}()
	}
	wg.Wait()
}

func BenchmarkConcurrent99Read1Write(b *testing.B) { benchConcurrent(b, 100) }
func BenchmarkConcurrent90Read10Write(b *testing.B) { benchConcurrent(b, 10) }
func BenchmarkConcurrent70Read30Write(b *testing.B) { benchConcurrent(b, 3) }
func BenchmarkConcurrent50Read50Write(b *testing.B) { benchConcurrent(b, 2) }
func BenchmarkConcurrent100Write(b *testing.B)    { benchConcurrent(b, 1) }
