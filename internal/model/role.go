package model

import (
	"errors"

	"github.com/gralliry/go-auther/internal/pkg/set"
)

var (
	ErrRoleInvalid          = errors.New("invalid role")
	ErrRoleAlreadyExist     = errors.New("role already exists")
	ErrRoleNotFound         = errors.New("role not found")
	ErrRoleInsufficient     = errors.New("insufficient role")
	ErrRoleInvalidHierarchy = errors.New("invalid role hierarchy")
	ErrRoleSelfGrant        = errors.New("self grant is not allowed")
)

// Role represents a node in the role tree.
type Role struct {
	// immutable field
	id string
	// Valid() verify this field is not nil
	// IsRoot() verify this field is self
	parent   *Role
	children *set.CacheSet[*Role]

	srcGrants *set.CacheSet[*Policy]
	tarGrants *set.CacheSet[*Policy]
}

func rawRole(id string) *Role {
	return &Role{
		id:        id,
		parent:    nil,
		children:  set.NewCacheSet[*Role](true),
		srcGrants: set.NewCacheSet[*Policy](false),
		tarGrants: set.NewCacheSet[*Policy](true),
	}
}

func (r *Role) ID() string {
	return r.id
}

func (r *Role) Valid() bool {
	return r != nil && r.parent != nil
}

func (r *Role) Parent() (*Role, error) {
	if !r.Valid() {
		return nil, ErrRoleInvalid
	}
	if r.parent == r {
		return nil, nil
	}
	return r.parent, nil
}

func (r *Role) IsParent(parent *Role) (bool, error) {
	if !r.Valid() {
		return false, ErrRoleInvalid
	}
	return r.parent == parent, nil
}

func (r *Role) Children() ([]*Role, error) {
	if !r.Valid() {
		return nil, ErrRoleInvalid
	}
	return r.children.ToSlice(), nil
}

func (r *Role) Enforce(resource Resource) (bool, error) {
	if !r.Valid() {
		return false, ErrRoleInvalid
	}
	return r.srcGrants.Any(func(p *Policy) bool {
		return p.Match(resource)
	}), nil
}

func (r *Role) Grant(grantee *Role, resource Resource) error {
	// check if r is valid
	if !r.Valid() {
		return ErrRoleInvalid
	}
	// check if grantee is valid
	if !grantee.Valid() {
		return ErrRoleInvalid
	}
	// check if grantee is an ancestor of r
	for p := r.parent; ; p = p.parent {
		if p == grantee {
			return ErrRoleInvalidHierarchy
		}
		if p == p.parent {
			break
		}
	}
	// check if r is grantee
	if r == grantee {
		return ErrRoleSelfGrant
	}
	// 检查是否有足够的权限
	if !r.srcGrants.Any(func(p *Policy) bool {
		return p.Match(resource)
	}) {
		return ErrRoleInsufficient
	}
	// Create policy
	policy := rawPolicy(0, r, grantee, resource)
	// Create policy if not exists
	if r.tarGrants.Any(func(p *Policy) bool {
		return p.Equal(policy)
	}) {
		return ErrPolicyAlreadyExist
	}
	// Add policy to target
	r.tarGrants.Add(policy)
	// Add policy to source
	grantee.srcGrants.Add(policy)
	return nil
}

func (r *Role) Revoke(policy *Policy) error {
	// check if policy is valid
	if !policy.Valid() {
		return ErrPolicyInvalid
	}
	// check policy exists
	if !r.tarGrants.Has(policy) {
		return ErrPolicyNotFound
	}
	// revoke policy
	policy.grantor = nil
	// clean target
	r.clean()
	return nil
}

// force clean all grants
// 最核心代码
func (r *Role) clean() {
	// 找到并清理无效的策略
	revoked := r.srcGrants.GC()
	// 找到需要审查的策略
	toReviewed := revoked.Filter(func(p1 *Policy) bool {
		// 授权的权限出现在了被撤销的策略中，可能无效，需要审查
		return r.tarGrants.Any(func(p2 *Policy) bool {
			return p1.Match(p2.resource)
		})
	})
	// 找到并清理需要撤销的策略
	toRevoked := toReviewed.ExtractIf(func(p1 *Policy) bool {
		// p1 没有出现 source 中的策略
		return !r.srcGrants.Any(func(p2 *Policy) bool {
			return p2.Match(p1.resource)
		})
	})
	// 开始递归撤销策略
	// 不要边遍历边递归，因为有可能多个权限指向同一个组，节约性能
	toRevoked.Range(func(p *Policy) {
		p.grantor = nil
	})
	// 清理 target 中的无效策略
	toRevoked.Range(func(p *Policy) {
		p.grantee.clean()
	})
}

func (r *Role) Received() ([]*Policy, error) {
	if !r.Valid() {
		return nil, ErrRoleInvalid
	}
	return r.srcGrants.ToSlice(), nil
}

func (r *Role) Granted() ([]*Policy, error) {
	if !r.Valid() {
		return nil, ErrRoleInvalid
	}
	return r.tarGrants.ToSlice(), nil
}
