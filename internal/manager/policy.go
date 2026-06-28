package manager

import (
	"github.com/gralliry/go-auther/internal/pkg/set"
	"github.com/gralliry/go-auther/internal/resource"
)

// Policy represents a single resource grant. It forms a node in a DAG:
// - parents: count of active parent policies (valid when > 0)
// - children: policies this one subsumes (linked via AutoCacheSet)
type Policy struct {
	id       int64
	resource *resource.Resource
	parents  int                   // active parent count; Valid() == parents > 0
	children *set.AutoCacheSet[*Policy] // child policies this one contains
}

// newPolicy creates an unlinked policy with the given id and resource.
// The caller is responsible for wiring parent/child DAG links.
func newPolicy(id int64, res *resource.Resource) *Policy {
	return &Policy{
		id:       id,
		children: set.NewAutoCacheSet[*Policy](),
		resource: res,
	}
}

// Valid reports whether the policy is still active.
func (p *Policy) Valid() bool { return p != nil && p.parents > 0 }

// contains reports whether this policy's resource subsumes target.
func (p *Policy) contains(target *resource.Resource) bool {
	return p.resource.Contains(target)
}

// within reports whether this policy's resource is subsumed by pattern.
func (p *Policy) within(pattern *resource.Resource) bool {
	return pattern.Contains(p.resource)
}

// match reports whether the given raw path string matches this policy's resource pattern.
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
