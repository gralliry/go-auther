package model

import (
	"errors"

	"github.com/gralliry/go-auther/internal/pkg/set"
)

var (
	ErrPolicyNotFound = errors.New("policy not found")
)

type Policy struct {
	id       int64
	res      Resource
	parents  int
	children *set.AutoCacheSet[*Policy]
	area     *Area
}

func newPolicy(id int64, res Resource, area *Area) *Policy {
	return &Policy{
		id:       id,
		children: set.NewAutoCacheSet[*Policy](),
		res:      res,
		area:     area,
	}
}

func (p *Policy) ID() int64        { return p.id }
func (p *Policy) Valid() bool      { return p != nil && (p.parents > 0 || p.id == 0) }
func (p *Policy) Resource() string { return string(p.res) }

func (p *Policy) contains(target Resource) bool {
	return p.res.Match(target)
}

func (p *Policy) within(pattern Resource) bool {
	return pattern.Match(p.res)
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
