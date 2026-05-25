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
	"fmt"
	"sync"

	"auther/internal/model"
	"auther/snapshot"
)

// GrantInfo 通过别名暴露给外部使用。
type GrantInfo = model.GrantInfo

// Authorizer 是权限系统的主入口，管理角色树、用户映射和资源授权。
type Authorizer struct {
	mu      sync.RWMutex
	root    *model.RoleNode
	roles   map[string]*model.RoleNode
	users   map[string]*model.UserNode
	adapter Adapter
}

// NewAuthorizer 使用给定的适配器创建 Authorizer。
// 如果适配器中已存储数据，则会加载恢复；否则自动创建一个
// ID 为 "root" 且拥有 "/**" 资源的根角色。
// adapter 可以为 nil，此时数据仅保存在内存中。
func NewAuthorizer(adapter Adapter) (*Authorizer, error) {
	a := &Authorizer{
		adapter: adapter,
		roles:   make(map[string]*model.RoleNode),
		users:   make(map[string]*model.UserNode),
	}

	var snap *snapshot.Policy
	if adapter != nil {
		var err error
		snap, err = adapter.Load()
		if err != nil {
			return nil, fmt.Errorf("auther: load policy: %w", err)
		}
	}
	if snap != nil && len(snap.Roles) > 0 {
		if err := a.buildTree(snap); err != nil {
			return nil, err
		}
		return a, nil
	}

	a.root = &model.RoleNode{
		ID:         "root",
		Children:   make(map[string]*model.RoleNode),
		GrantedMap: map[string]bool{"/**": true},
		Users:      make(map[string]*model.UserNode),
	}
	a.roles["root"] = a.root

	if err := a.save(); err != nil {
		return nil, fmt.Errorf("auther: save initial state: %w", err)
	}
	return a, nil
}

// buildTree 从持久化快照重建内存中的角色树。
// 在加载过程中会自动修复损坏的数据（孤立角色、悬挂用户、无效授权等），
// 并在确实清理了数据时才将修复后的状态写回适配器。
func (a *Authorizer) buildTree(snapshot *snapshot.Policy) error {
	a.roles = make(map[string]*model.RoleNode)
	a.users = make(map[string]*model.UserNode)

	rootID, cleansed := a.buildRoles(snapshot.Roles)
	cleansed = a.linkParents(snapshot.Roles, rootID) || cleansed

	if err := a.checkCycle(); err != nil {
		return err
	}

	cleansed = a.loadUsers(snapshot.Users) || cleansed
	cleansed = a.loadGrants(snapshot.Grants) || cleansed

	if cleansed && a.adapter != nil {
		if err := a.adapter.Save(a.snapshot()); err != nil {
			return fmt.Errorf("auther: persist cleansed state: %w", err)
		}
	}
	return nil
}

// buildRoles 创建所有角色节点，返回根角色 ID 和是否发生过清理。
func (a *Authorizer) buildRoles(roles []snapshot.Role) (rootID string, cleansed bool) {
	for _, rs := range roles {
		role := &model.RoleNode{
			ID:         rs.ID,
			Children:   make(map[string]*model.RoleNode),
			GrantedMap: make(map[string]bool),
			Users:      make(map[string]*model.UserNode),
		}
		a.roles[rs.ID] = role
	}

	// 确定根角色：首个 ParentID 为空的角色。
	for _, rs := range roles {
		if rs.ParentID == "" {
			rootID = rs.ID
			break
		}
	}
	if rootID == "" {
		cleansed = true
		rootID = "root"
		if a.roles["root"] == nil {
			a.roles["root"] = &model.RoleNode{
				ID:         "root",
				Children:   make(map[string]*model.RoleNode),
				GrantedMap: map[string]bool{"/**": true},
				Users:      make(map[string]*model.UserNode),
			}
		}
	}
	a.root = a.roles[rootID]
	return rootID, cleansed
}

// linkParents 为所有角色建立父子链接。无效父角色 → 挂载到根。
func (a *Authorizer) linkParents(roles []snapshot.Role, rootID string) (cleansed bool) {
	for _, rs := range roles {
		if rs.ID == rootID {
			continue
		}
		role := a.roles[rs.ID]
		if role == nil {
			continue
		}
		parent := a.roles[rs.ParentID]
		if parent == nil || rs.ParentID == "" {
			cleansed = true
			parent = a.root
		}
		role.Parent = parent
		parent.Children[rs.ID] = role
	}
	return cleansed
}

// loadUsers 加载用户快照。所属角色不存在 → 丢弃。
func (a *Authorizer) loadUsers(users []snapshot.User) (cleansed bool) {
	for _, us := range users {
		role := a.roles[us.RoleID]
		if role == nil {
			cleansed = true
			continue
		}
		user := &model.UserNode{ID: us.ID, Role: role}
		a.users[us.ID] = user
		role.Users[us.ID] = user
	}
	return cleansed
}

// loadGrants 加载授权记录。无效授权、重复授权、自授权（转为 GrantedMap 条目）均被清洗。
func (a *Authorizer) loadGrants(grants []snapshot.Grant) (cleansed bool) {
	seen := make(map[string]bool)
	for _, gs := range grants {
		fromRole := a.roles[gs.FromRoleID]
		toRole := a.roles[gs.ToRoleID]
		if fromRole == nil || toRole == nil {
			cleansed = true
			continue
		}

		// 自授权：将资源直接加入 GrantedMap，不创建 GrantInfo 记录。
		if gs.FromRoleID == gs.ToRoleID {
			cleansed = true
			toRole.GrantedMap[gs.Resource] = true
			continue
		}

		if !a.isAncestor(gs.FromRoleID, gs.ToRoleID) {
			cleansed = true
			continue
		}
		key := gs.FromRoleID + "|" + gs.ToRoleID + "|" + gs.Resource
		if seen[key] {
			cleansed = true
			continue
		}
		seen[key] = true

		grant := model.GrantInfo{FromRoleID: gs.FromRoleID, ToRoleID: gs.ToRoleID, Resource: gs.Resource}
		fromRole.GrantsOut = append(fromRole.GrantsOut, grant)
		toRole.GrantsIn = append(toRole.GrantsIn, grant)
		toRole.GrantedMap[gs.Resource] = true
	}
	return cleansed
}

// snapshot 将当前内存中的角色树转换为可序列化的策略快照。
func (a *Authorizer) snapshot() *snapshot.Policy {
	snap := &snapshot.Policy{}

	var walk func(role *model.RoleNode)
	walk = func(role *model.RoleNode) {
		rs := snapshot.Role{ID: role.ID}
		if role.Parent != nil {
			rs.ParentID = role.Parent.ID
		}
		snap.Roles = append(snap.Roles, rs)
		for _, user := range role.Users {
			snap.Users = append(snap.Users, snapshot.User{ID: user.ID, RoleID: user.Role.ID})
		}
		for _, child := range role.Children {
			walk(child)
		}
	}
	walk(a.root)

	seen := make(map[string]bool)
	var collectGrants func(role *model.RoleNode)
	collectGrants = func(role *model.RoleNode) {
		for _, g := range role.GrantsOut {
			key := g.FromRoleID + "|" + g.ToRoleID + "|" + g.Resource
			if !seen[key] {
				seen[key] = true
				snap.Grants = append(snap.Grants, snapshot.Grant{
					FromRoleID: g.FromRoleID, ToRoleID: g.ToRoleID, Resource: g.Resource,
				})
			}
		}
		for _, child := range role.Children {
			collectGrants(child)
		}
	}
	collectGrants(a.root)
	return snap
}

// save 将当前状态全量写入适配器。
func (a *Authorizer) save() error {
	if a.adapter == nil {
		return nil
	}
	return a.adapter.Save(a.snapshot())
}

// saveSetRole 持久化角色创建。
func (a *Authorizer) saveSetRole(roleID, parentID string) error {
	if a.adapter == nil {
		return nil
	}
	return a.adapter.SetRole(snapshot.Role{ID: roleID, ParentID: parentID})
}

// saveSetUser 持久化用户创建。
func (a *Authorizer) saveSetUser(roleID, userID string) error {
	if a.adapter == nil {
		return nil
	}
	return a.adapter.SetUser(snapshot.User{ID: userID, RoleID: roleID})
}

// saveUnsetUser 持久化用户删除。
func (a *Authorizer) saveUnsetUser(roleID, userID string) error {
	if a.adapter == nil {
		return nil
	}
	return a.adapter.UnsetUser(snapshot.User{ID: userID, RoleID: roleID})
}

// saveSetGrant 持久化授权添加。
func (a *Authorizer) saveSetGrant(fromRoleID, toRoleID, resource string) error {
	if a.adapter == nil {
		return nil
	}
	return a.adapter.SetGrant(snapshot.Grant{FromRoleID: fromRoleID, ToRoleID: toRoleID, Resource: resource})
}

// subtree 收集指定角色及其所有后代角色，使用 BFS 遍历。
func (a *Authorizer) subtree(roleID string) []*model.RoleNode {
	role := a.roles[roleID]
	if role == nil {
		return nil
	}
	var result []*model.RoleNode
	queue := []*model.RoleNode{role}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		result = append(result, cur)
		for _, child := range cur.Children {
			queue = append(queue, child)
		}
	}
	return result
}

// isAncestor 判断 ancestorID 是否为 descendantID 的祖先角色。
func (a *Authorizer) isAncestor(ancestorID, descendantID string) bool {
	for d := a.roles[descendantID]; d != nil; d = d.Parent {
		if d.ID == ancestorID {
			return true
		}
	}
	return false
}

// checkCycle 检测角色树中是否存在循环引用，O(n) 时间 O(n) 空间。
func (a *Authorizer) checkCycle() error {
	verified := make(map[string]bool)
	for _, role := range a.roles {
		path := make(map[string]bool)
		for cur := role; cur != nil; cur = cur.Parent {
			if path[cur.ID] {
				return fmt.Errorf("%w: detected at role %s", ErrCircularRoleHierarchy, role.ID)
			}
			if verified[cur.ID] {
				break
			}
			path[cur.ID] = true
			verified[cur.ID] = true
		}
	}
	return nil
}
