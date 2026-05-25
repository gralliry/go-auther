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
	"sort"
	"sync"

	"auther/model"
)

// 类型别名：外部只需 import "auther" 即可使用以下类型。
type (
	Resource       = model.Resource
	GrantInfo      = model.GrantInfo
	RoleInfo       = model.RoleInfo
	UserInfo       = model.UserInfo
	PolicySnapshot = model.PolicySnapshot
	RoleSnapshot   = model.RoleSnapshot
	UserSnapshot   = model.UserSnapshot
	GrantSnapshot  = model.GrantSnapshot
)

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

	var snap *model.PolicySnapshot
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
		Resources:  map[Resource]bool{"/**": true},
		GrantedMap: make(map[Resource]bool),
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
func (a *Authorizer) buildTree(snapshot *model.PolicySnapshot) error {
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
func (a *Authorizer) buildRoles(roles []model.RoleSnapshot) (rootID string, cleansed bool) {
	for _, rs := range roles {
		role := &model.RoleNode{
			ID:         rs.ID,
			Children:   make(map[string]*model.RoleNode),
			Resources:  make(map[Resource]bool),
			GrantedMap: make(map[Resource]bool),
			Users:      make(map[string]*model.UserNode),
		}
		for _, res := range rs.Resources {
			role.Resources[res] = true
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
				Resources:  map[Resource]bool{"/**": true},
				GrantedMap: make(map[Resource]bool),
				Users:      make(map[string]*model.UserNode),
			}
		}
	}
	a.root = a.roles[rootID]
	return rootID, cleansed
}

// linkParents 为所有角色建立父子链接。无效父角色 → 挂载到根。
func (a *Authorizer) linkParents(roles []model.RoleSnapshot, rootID string) (cleansed bool) {
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
func (a *Authorizer) loadUsers(users []model.UserSnapshot) (cleansed bool) {
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

// loadGrants 加载授权记录。无效授权、重复授权、自授权均被清洗。
func (a *Authorizer) loadGrants(grants []model.GrantSnapshot) (cleansed bool) {
	seen := make(map[string]bool)
	for _, gs := range grants {
		fromRole := a.roles[gs.FromRoleID]
		toRole := a.roles[gs.ToRoleID]
		if fromRole == nil || toRole == nil {
			cleansed = true
			continue
		}
		if gs.FromRoleID == gs.ToRoleID {
			cleansed = true
			toRole.Resources[gs.Resource] = true
			continue
		}
		if !a.isAncestorOrSelf(gs.FromRoleID, gs.ToRoleID) {
			cleansed = true
			continue
		}
		key := gs.FromRoleID + "|" + gs.ToRoleID + "|" + string(gs.Resource)
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
func (a *Authorizer) snapshot() *model.PolicySnapshot {
	snap := &model.PolicySnapshot{}

	var walk func(role *model.RoleNode)
	walk = func(role *model.RoleNode) {
		rs := model.RoleSnapshot{
			ID:        role.ID,
			Resources: make([]Resource, 0, len(role.Resources)),
		}
		if role.Parent != nil {
			rs.ParentID = role.Parent.ID
		}
		for res := range role.Resources {
			rs.Resources = append(rs.Resources, res)
		}
		sort.Slice(rs.Resources, func(i, j int) bool { return rs.Resources[i] < rs.Resources[j] })
		snap.Roles = append(snap.Roles, rs)
		for _, user := range role.Users {
			snap.Users = append(snap.Users, model.UserSnapshot{ID: user.ID, RoleID: user.Role.ID})
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
			key := g.FromRoleID + "|" + g.ToRoleID + "|" + string(g.Resource)
			if !seen[key] {
				seen[key] = true
				snap.Grants = append(snap.Grants, model.GrantSnapshot{
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

// save 将当前状态通过适配器持久化。如果适配器为空则直接返回。
func (a *Authorizer) save() error {
	if a.adapter == nil {
		return nil
	}
	return a.adapter.Save(a.snapshot())
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

// isAncestorOrSelf 判断 aID 是否是 dID 的祖先或其自身。
func (a *Authorizer) isAncestorOrSelf(aID, dID string) bool {
	return aID == dID || a.isAncestor(aID, dID)
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
