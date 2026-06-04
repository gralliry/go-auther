package model

import (
	"errors"

	"github.com/gralliry/go-auther/internal/pkg/set"
)

var (
	ErrPolicyInvalid  = errors.New("policy invalid")
	ErrPolicyNotMatch = errors.New("policy not match")
	ErrPolicyNotFound = errors.New("policy not found")
)

type Policy struct {
	id       int64
	res      Resource
	parents  *set.AutoCacheSet[*Policy]
	children *set.AutoCacheSet[*Policy]
	valid    bool
	area     *Area
}

func newPolicy(id int64, res Resource, area *Area) *Policy {
	return &Policy{
		id:       id,
		parents:  set.NewAutoCacheSet[*Policy](),
		children: set.NewAutoCacheSet[*Policy](),
		res:      res,
		valid:    true,
		area:     area,
	}
}

func (p *Policy) ID() int64     { return p.id }
func (p *Policy) Valid() bool   { return p != nil && p.valid }
func (p *Policy) Resource() string { return string(p.res) }

func (p *Policy) contains(target Resource) bool {
	return p.res.Match(target)
}

func (p *Policy) within(pattern Resource) bool {
	return pattern.Match(p.res)
}

// revoke invalidates this policy and cascades to orphaned children.
// The caller must hold p.area.mutex (write lock).
func (p *Policy) revoke() {
	if !p.Valid() {
		return
	}
	p.valid = false
	p.area.DeletePolicy(p.id)
	p.parents.Range(func(parent *Policy) {
		parent.children.Delete(p)
	})
	p.children.Range(func(child *Policy) {
		child.parents.Delete(p)
		if !child.parents.Any(func(pp *Policy) bool {
			return pp.Valid()
		}) {
			child.revoke()
		}
	})
}
