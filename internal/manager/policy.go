package manager

import (
	"github.com/gralliry/go-auther/internal/pkg/set"
	"github.com/gralliry/go-auther/internal/resource"
)

// ---------------------------------------------------------------------------
// Policy type
// ---------------------------------------------------------------------------

type Policy struct {
	id       int64
	resource *resource.Resource
	parents  int
	children *set.AutoCacheSet[*Policy]
}

func newPolicy(id int64, res *resource.Resource) *Policy {
	return &Policy{
		id:       id,
		children: set.NewAutoCacheSet[*Policy](),
		resource: res,
	}
}

// Valid reports whether the policy is still active.
func (p *Policy) Valid() bool { return p != nil && p.parents > 0 }

// ---------------------------------------------------------------------------
// Policy methods
// ---------------------------------------------------------------------------

func (p *Policy) contains(target *resource.Resource) bool {
	return p.resource.Contains(target)
}

func (p *Policy) within(pattern *resource.Resource) bool {
	return pattern.Contains(p.resource)
}

func (p *Policy) match(pattern string) bool {
	return p.resource.Match(pattern)
}

// revoke invalidates this policy and cascades to orphaned children.
// The onDelete callback is called for each policy that is invalidated.
func (p *Policy) revoke(onDelete func(int64)) {
	if p.parents < 0 {
		return
	}
	p.parents = -1
	if onDelete != nil {
		onDelete(p.id)
	}
	p.children.Range(func(child *Policy) {
		child.parents--
		if child.parents == 0 {
			child.revoke(onDelete)
		}
	})
}
