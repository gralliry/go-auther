package auther

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/gralliry/go-auther/internal/resource"
)

var (
	benchMatchResult bool
	benchGrantResult error
)

func BenchmarkMatchExact(b *testing.B) {
	for b.Loop() {
		benchMatchResult = resource.Resource("/user/create").Match("/user/create")
	}
}

func BenchmarkMatchNoMatchLiteral(b *testing.B) {
	for b.Loop() {
		benchMatchResult = resource.Resource("/user/create").Match("/user/delete")
	}
}

func BenchmarkMatchStar(b *testing.B) {
	for b.Loop() {
		benchMatchResult = resource.Resource("/user/*/edit").Match("/user/123/edit")
	}
}

func BenchmarkMatchDoubleStar(b *testing.B) {
	for b.Loop() {
		benchMatchResult = resource.Resource("/a/**/z").Match("/a/b/c/d/e/z")
	}
}

func BenchmarkMatchDoubleStarMany(b *testing.B) {
	for b.Loop() {
		benchMatchResult = resource.Resource("/api/**/export").Match("/api/v1/users/admin/reports/2024/export")
	}
}

// =============================================================================
// Enforce benchmarks — full enforcement path (role lookup + resource matching)
// =============================================================================

func benchAuthorizer(b *testing.B) *Authorizer {
	b.Helper()
	a, _ := NewAuthorizer(&testAdapter{})
	_ = a.CreateRole("root", "admin")
	_ = a.CreateRole("admin", "editor")
	_ = a.Grant("root", "admin", "/user/*")
	_ = a.Grant("root", "editor", "/data/*")
	_ = a.Grant("root", "admin", "/g/**")
	_ = a.CreateUser("admin", "u_admin")
	_ = a.CreateUser("editor", "u_editor")
	return a
}

// Hit: exact match on role's own resource (fast path, no wildcard).
func BenchmarkEnforceHitExact(b *testing.B) {
	a := benchAuthorizer(b)
	b.ResetTimer()
	for b.Loop() {
		a.Enforce("u_editor", "/data/read")
	}
}

// Hit: wildcard match on role's own resource (needs segment parse + DP).
func BenchmarkEnforceHitWildcard(b *testing.B) {
	a := benchAuthorizer(b)
	b.ResetTimer()
	for b.Loop() {
		a.Enforce("u_admin", "/user/edit")
	}
}

// Hit: match via grant (grant patterns checked after own resources).
func BenchmarkEnforceHitGrant(b *testing.B) {
	a := benchAuthorizer(b)
	b.ResetTimer()
	for b.Loop() {
		a.Enforce("u_admin", "/g/anything/here")
	}
}

// Miss: no wildcards to check, first literal comparison fails immediately.
func BenchmarkEnforceMissLiteral(b *testing.B) {
	a := benchAuthorizer(b)
	b.ResetTimer()
	for b.Loop() {
		a.Enforce("u_editor", "/user/create")
	}
}

// Miss: scans all patterns (2 own + 1 grant) and all fail — worst case.
func BenchmarkEnforceMissAll(b *testing.B) {
	a := benchAuthorizer(b)
	b.ResetTimer()
	for b.Loop() {
		a.Enforce("u_editor", "/nonexistent/path")
	}
}

// =============================================================================
// Grant / Revoke benchmarks — permission modification operations
// =============================================================================

// BenchmarkGrant measures a single Grant call (acquire write lock, validate,
// link grant nodes, persist) on a shallow hierarchy.
func BenchmarkGrant(b *testing.B) {
	a, _ := NewAuthorizer(&testAdapter{})
	_ = a.CreateRole("root", "target")
	b.ResetTimer()
	for b.Loop() {
		benchGrantResult = a.Grant("root", "target", "/resource/*")
	}
}

// BenchmarkGrantToDeepRole measures Grant where the grantee is 10 levels deep,
// so HasAncestor traverses the full chain before granting.
func BenchmarkGrantToDeepRole(b *testing.B) {
	a, _ := NewAuthorizer(&testAdapter{})
	parent := "root"
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("r%d", i)
		_ = a.CreateRole(parent, id)
		parent = id
	}
	b.ResetTimer()
	for b.Loop() {
		benchGrantResult = a.Grant("root", "r9", "/deep/resource/*")
	}
}

// BenchmarkRevoke measures a single Revoke call (find, remove, persist).
// Re-grants inside the loop so each iteration starts fresh.
func BenchmarkRevoke(b *testing.B) {
	a, _ := NewAuthorizer(&testAdapter{})
	_ = a.CreateRole("root", "target")
	b.ResetTimer()
	for b.Loop() {
		_ = a.Grant("root", "target", "/resource/*")
		benchGrantResult = a.Revoke("root", "target", "/resource/*")
	}
}

// BenchmarkRevokeCascade measures Revoke with cascade cleanup through a
// 3-level delegation chain (root→r1→r2→r3, each granting the same resource).
func BenchmarkRevokeCascade(b *testing.B) {
	a, _ := NewAuthorizer(&testAdapter{})
	_ = a.CreateRole("root", "r1")
	_ = a.CreateRole("r1", "r2")
	_ = a.CreateRole("r2", "r3")
	b.ResetTimer()
	for b.Loop() {
		_ = a.Grant("root", "r1", "/cascade")
		_ = a.Grant("r1", "r2", "/cascade")
		_ = a.Grant("r2", "r3", "/cascade")
		benchGrantResult = a.Revoke("root", "r1", "/cascade")
	}
}

// =============================================================================
// Concurrent benchmarks — mixed read (Enforce) and write (Grant/Revoke)
// =============================================================================

// benchConcurrentMix runs parallel goroutines mixing Enforce (RLock) with
// Grant+Revoke (Lock). Each goroutine independently decides read vs write
// using the given write probability, producing truly random interleaving.
func benchConcurrentMix(b *testing.B, writeProb float64) {
	a := benchAuthorizer(b)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		for pb.Next() {
			if rng.Float64() < writeProb {
				a.Grant("root", "admin", "/tmp")
				a.Revoke("root", "admin", "/tmp")
			} else {
				a.Enforce("u_admin", "/user/edit")
			}
		}
	})
}

// BenchmarkConcurrentRead99Write1: 99% reads, 1% writes.
func BenchmarkConcurrentRead99Write1(b *testing.B) { benchConcurrentMix(b, 0.01) }

// BenchmarkConcurrentRead90Write10: 90% reads, 10% writes.
func BenchmarkConcurrentRead90Write10(b *testing.B) { benchConcurrentMix(b, 0.10) }

// BenchmarkConcurrentRead80Write20: 80% reads, 20% writes.
func BenchmarkConcurrentRead80Write20(b *testing.B) { benchConcurrentMix(b, 0.20) }

// BenchmarkConcurrentRead70Write30: 70% reads, 30% writes.
func BenchmarkConcurrentRead70Write30(b *testing.B) { benchConcurrentMix(b, 0.30) }

// BenchmarkConcurrentRead50Write50: 50% reads, 50% writes.
func BenchmarkConcurrentRead50Write50(b *testing.B) { benchConcurrentMix(b, 0.50) }

// BenchmarkConcurrentAllWrite: 100% writes — pure write lock contention.
func BenchmarkConcurrentAllWrite(b *testing.B) { benchConcurrentMix(b, 1.0) }
