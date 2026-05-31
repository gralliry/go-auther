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
	// immutable field
	id int64

	resource Resource

	// when grantable == false, children must be nil or empty
	children *set.AutoCacheSet[*Policy]

	valid bool
}

func newPolicy(id int64, resource Resource) *Policy {
	return &Policy{
		id:       id,
		children: set.NewAutoCacheSet[*Policy](),
		resource: resource,
		valid:    true,
	}
}

func (p *Policy) ID() int64 {
	return p.id
}

func (p *Policy) Valid() bool {
	return p != nil && p.valid
}

func (p *Policy) Match(resource Resource) bool {
	return p.resource.Match(resource)
}

func (p *Policy) delegate(resource Resource) (*Policy, error) {
	if !p.Valid() {
		return nil, ErrPolicyInvalid
	}
	if !p.Match(resource) {
		return nil, ErrPolicyNotMatch
	}
	policy := &Policy{
		id:       node.Generate().Int64(),
		children: set.NewAutoCacheSet[*Policy](),
		resource: resource,
		valid:    true,
	}
	p.children.Add(policy)
	return policy, nil
}

func (p *Policy) revoke() {
	// 断开与 parent 的关联
	p.valid = false
	// 递归撤销子策略
	p.children.Range(func(child *Policy) {
		child.revoke()
	})
}
