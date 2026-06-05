package model

import (
	"github.com/gralliry/go-auther/internal/pkg/set"
)

type Policy struct {
	id       int64
	resource *Resource
	parents  int
	children *set.AutoCacheSet[*Policy]
	area     *Area
}

func newPolicy(id int64, resource *Resource, area *Area) *Policy {
	return &Policy{
		id:       id,
		children: set.NewAutoCacheSet[*Policy](),
		resource: resource,
		area:     area,
	}
}

// ID returns the policy's unique snowflake identifier.
func (p *Policy) ID() int64 { return p.id }

// Valid reports whether the policy is still active. A policy is valid if it has
// at least one active parent, or if it is the root policy (id == 0).
func (p *Policy) Valid() bool { return p != nil && p.parents > 0 }

// Resource returns the resource pattern this policy grants.
func (p *Policy) Resource() string { return p.resource.String() }

func (p *Policy) contains(target *Resource) bool {
	return p.resource.Match(target.raw)
}

func (p *Policy) within(pattern *Resource) bool {
	return pattern.Match(p.resource.raw)
}

func (p *Policy) match(pattern string) bool {
	return p.resource.Match(pattern)
}

// revoke invalidates this policy and cascades to orphaned children.
// The caller must hold area write lock.
func (p *Policy) revoke() {
	if p.parents < 0 {
		return
	}
	p.parents = -1
	p.area.DeletePolicy(p.id)
	p.children.Range(func(child *Policy) {
		child.parents--
		if child.parents == 0 {
			child.revoke()
		}
	})
}
