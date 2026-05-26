// Package auther 提供基于角色树的权限管理库。
//
// 核心概念
//
// Auther 管理三个核心实体：
//
//   - 角色（Role）：形成树形层级结构。根角色在初始化时自动创建，
//     并默认拥有 "/**" 资源权限。角色可以创建子角色和用户。
//     权限不会自动继承 —— 父角色必须显式调用 Grant 向子角色授权。
//
//   - 用户（User）：由角色创建的被动叶子节点。用户继承其所属角色的
//     有效权限，但不能管理资源或创建其他用户/角色。
//
//   - 资源（Resource）：路径风格的字符串（例如 /user/create、/data/**）。
//     支持 glob 匹配：* 匹配单个路径段，** 匹配零个或多个路径段。
//
// 持久化
//
// 适配器（Adapter）提供持久化能力。每次数据变更立即写透到适配器中。
// 构造 Authorizer 时，如果适配器中存在已持久化的数据，会自动加载恢复状态。
//
// 并发安全
//
// Authorizer 的所有公开方法均受 sync.RWMutex 保护，可安全并发使用。
package auther

import (
	"sync"

	"github.com/gralliry/go-auther/internal/model"
)

// GrantNode 通过别名暴露给外部使用。
type GrantNode = model.GrantNode

// RoleInfo 通过别名暴露给外部使用。
type RoleInfo = model.RoleInfo

// Authorizer 是权限系统的主入口，管理角色树、用户映射和资源授权。
type Authorizer struct {
	mu      sync.RWMutex
	roles   map[string]*model.RoleNode
	users   map[string]*model.UserNode
	adapter Adapter
}

// NewAuthorizer 使用给定的适配器创建 Authorizer。
// 如果适配器中已存储数据，则会加载恢复；否则自动创建一个
// ID 为 "root" 且拥有 "/**" 资源的根角色。
// adapter 不能为 nil。
func NewAuthorizer(adapter Adapter) (*Authorizer, error) {
	if adapter == nil {
		return nil, ErrAdapterRequired
	}
	a := &Authorizer{
		adapter: adapter,
		roles:   make(map[string]*model.RoleNode),
		users:   make(map[string]*model.UserNode),
	}
	if err := a.Load(); err != nil {
		return nil, err
	}
	if a.roles["root"] == nil {
		a.roles["root"] = &model.RoleNode{
			ID:         "root",
			Children:   make(map[string]*model.RoleNode),
			GrantedMap: map[string]bool{"/**": true},
			Users:      make(map[string]*model.UserNode),
		}
		if err := a.save(); err != nil {
			return nil, err
		}
	}
	return a, nil
}
