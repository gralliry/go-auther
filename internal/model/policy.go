package model

import "errors"

var (
	ErrPolicyInvalid      = errors.New("invalid policy")
	ErrPolicyNotFound     = errors.New("policy not found")
	ErrPolicyAlreadyExist = errors.New("policy already exists")
)

// Policy represents an explicit resource grant from an ancestor role to a descendant.
type Policy struct {
	// immutable field
	id int64
	// Valid() verify grantor and grantee is not nil
	grantor  *Role
	grantee  *Role
	resource Resource
}

func rawPolicy(id int64, grantor *Role, grantee *Role, resource Resource) *Policy {
	return &Policy{
		id:       id,
		grantor:  grantor,
		grantee:  grantee,
		resource: resource,
	}
}

func (p *Policy) ID() int64 {
	return p.id
}

func (p *Policy) Valid() bool {
	return p != nil && p.grantor != nil && p.grantee != nil
}

func (p *Policy) Grantor() (*Role, error) {
	if !p.Valid() {
		return nil, ErrPolicyInvalid
	}
	return p.grantor, nil
}

func (p *Policy) Grantee() (*Role, error) {
	if !p.Valid() {
		return nil, ErrPolicyInvalid
	}
	return p.grantee, nil
}

func (p *Policy) Match(resource Resource) bool {
	// reduce function call
	return p.resource.Match(resource)
}

func (p *Policy) Equal(p2 *Policy) bool {
	return p.Valid() && p2.Valid() &&
		p.grantee == p2.grantee &&
		p.grantor == p2.grantor &&
		p.resource == p2.resource
}
