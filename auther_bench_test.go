package auther

import (
	"strconv"
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
	t := NewResource("/user/create")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool = r.Match(t.String())
	}
}

func BenchmarkMatchLiteralMiss(b *testing.B) {
	r := NewResource("/user/create")
	t := NewResource("/user/delete")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool = r.Match(t.String())
	}
}

func BenchmarkMatchSingleStar(b *testing.B) {
	r := NewResource("/user/*")
	t := NewResource("/user/create")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool = r.Match(t.String())
	}
}

func BenchmarkMatchDoubleStar(b *testing.B) {
	r := NewResource("/data/**")
	t := NewResource("/data/a/b/c")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool = r.Match(t.String())
	}
}

func BenchmarkMatchDeepDoubleStar(b *testing.B) {
	r := NewResource("/api/v1/**")
	t := NewResource("/api/v1/users/admin/permissions/read/write")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool = r.Match(t.String())
	}
}

// ---------------------------------------------------------------------------
// Create + Match (1:1 ratio)
// ---------------------------------------------------------------------------

func BenchmarkCreateMatchExact(b *testing.B) {
	for i := 0; i < b.N; i++ {
		r := NewResource("/user/create")
		t := NewResource("/user/create")
		benchBool = r.Match(t.String())
	}
}

func BenchmarkCreateMatchWildcard(b *testing.B) {
	for i := 0; i < b.N; i++ {
		r := NewResource("/user/*/edit")
		t := NewResource("/user/alice/edit")
		benchBool = r.Match(t.String())
	}
}

func BenchmarkCreateMatchDoubleStar(b *testing.B) {
	for i := 0; i < b.N; i++ {
		r := NewResource("/api/v1/**")
		t := NewResource("/api/v1/users/admin/data")
		benchBool = r.Match(t.String())
	}
}

// ---------------------------------------------------------------------------
// MatchString vs MatchResource
// ---------------------------------------------------------------------------

func BenchmarkMatchStringExact(b *testing.B) {
	r := NewResource("/user/create")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool = r.Match("/user/create")
	}
}

func BenchmarkMatchStringWildcard(b *testing.B) {
	r := NewResource("/user/*/edit")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool = r.Match("/user/alice/edit")
	}
}

func BenchmarkMatchStringDoubleStar(b *testing.B) {
	r := NewResource("/api/v1/**")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool = r.Match("/api/v1/users/admin/data")
	}
}

// ---------------------------------------------------------------------------
// Enforcement
// ---------------------------------------------------------------------------

func BenchmarkEnforceExactHit(b *testing.B) {
	m, _ := NewManager(empty.New())
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")
	root.Grant(NewResource("/user/create"), admin)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool, _ = admin.Enforce("/user/create")
	}
}

func BenchmarkEnforceWildcardHit(b *testing.B) {
	m, _ := NewManager(empty.New())
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")
	root.Grant(NewResource("/user/*"), admin)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool, _ = admin.Enforce("/user/create")
	}
}

func BenchmarkEnforceLiteralMiss(b *testing.B) {
	m, _ := NewManager(empty.New())
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")
	root.Grant(NewResource("/user/*"), admin)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool, _ = admin.Enforce("/data/read")
	}
}

func BenchmarkEnforceRoot(b *testing.B) {
	m, _ := NewManager(empty.New())
	root, _ := m.GetRole("root")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool, _ = root.Enforce("/anything")
	}
}

func BenchmarkEnforceUser(b *testing.B) {
	m, _ := NewManager(empty.New())
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")
	root.Grant(NewResource("/user/*"), admin)
	alice, _ := m.CreateUser("alice")
	alice.Assign(admin)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool, _ = alice.Enforce("/user/create")
	}
}

func BenchmarkEnforceManyPolicies(b *testing.B) {
	m, _ := NewManager(empty.New())
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")
	for i := 0; i < 20; i++ {
		id := string(rune('a' + i))
		root.Grant(NewResource("/"+id+"/*"), admin)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool, _ = admin.Enforce("/k/xyz")
	}
}

// ---------------------------------------------------------------------------
// Mutation
// ---------------------------------------------------------------------------

func BenchmarkGrant(b *testing.B) {
	m, _ := NewManager(empty.New())
	root, _ := m.GetRole("root")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := string(rune('A'+i%26)) + strconv.Itoa(i/26)
		editor, _ := m.CreateRole(id)
		root.Grant(NewResource("/data/*"), editor)
	}
}

func BenchmarkRevoke(b *testing.B) {
	m, _ := NewManager(empty.New())
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
	m, _ := NewManager(empty.New())
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

func BenchmarkRoleDelete(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		m, _ := NewManager(empty.New())
		root, _ := m.GetRole("root")
		admin, _ := m.CreateRole("admin")
		editor, _ := m.CreateRole("editor")
		root.Grant(NewResource("/user/*"), admin)
		admin.Grant(NewResource("/user/profile"), editor)
		b.StartTimer()

		admin.Delete()
	}
}

func BenchmarkUserAssign(b *testing.B) {
	m, _ := NewManager(empty.New())
	admin, _ := m.CreateRole("admin")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := string(rune('A'+i%26)) + strconv.Itoa(i/26)
		alice, _ := m.CreateUser(id)
		alice.Assign(admin)
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

// ---------------------------------------------------------------------------
// Concurrent read-write contention
// ---------------------------------------------------------------------------

func benchConcurrent(b *testing.B, readsPerWrite int) {
	m, _ := NewManager(empty.New())
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
					r, _ := m.CreateRole(string(rune('A'+id%26)) + strconv.Itoa(int(id/26)))
					root.Grant(NewResource("/data/*"), r)
				} else {
					alice.Enforce("/user/create")
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
