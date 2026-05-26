package auther

import (
	"fmt"

	"github.com/gralliry/go-auther/internal/model"
	"github.com/gralliry/go-auther/internal/resource"
	"github.com/gralliry/go-auther/snapshot"
)

// Load 从适配器加载数据并重建角色树，自动修复损坏数据后写回。
func (a *Authorizer) Load() error {
	snap, err := a.adapter.Load()
	if err != nil {
		return fmt.Errorf("auther: load policy: %w", err)
	}
	if snap == nil || len(snap.Roles) == 0 {
		return nil
	}

	cleansed := false
	a.loadRoles(snap)
	if a.ensureRoot() {
		cleansed = true
	}
	if a.linkRoleHierarchy(snap) {
		cleansed = true
	}
	if err := a.verifyNoCycles(); err != nil {
		return err
	}
	if a.loadUsers(snap) {
		cleansed = true
	}
	if a.loadGrants(snap) {
		cleansed = true
	}

	if cleansed {
		if err := a.save(); err != nil {
			return fmt.Errorf("auther: persist cleansed state: %w", err)
		}
	}
	return nil
}

func (a *Authorizer) loadRoles(snap *snapshot.Policy) {
	for _, rs := range snap.Roles {
		a.roles[rs.ID] = &model.RoleNode{
			ID:         rs.ID,
			Children:   make(map[string]*model.RoleNode),
			GrantedMap: make(map[string]bool),
			Users:      make(map[string]*model.UserNode),
		}
	}
}

func (a *Authorizer) ensureRoot() (repaired bool) {
	if a.roles["root"] != nil {
		return false
	}
	a.roles["root"] = &model.RoleNode{
		ID:         "root",
		Children:   make(map[string]*model.RoleNode),
		GrantedMap: map[string]bool{"/**": true},
		Users:      make(map[string]*model.UserNode),
	}
	return true
}

func (a *Authorizer) linkRoleHierarchy(snap *snapshot.Policy) (repaired bool) {
	root := a.roles["root"]
	for _, rs := range snap.Roles {
		if rs.ID == "root" {
			continue
		}
		parent := a.roles[rs.ParentID]
		if parent == nil {
			repaired = true
			parent = root
		}
		role := a.roles[rs.ID]
		role.Parent = parent
		parent.Children[rs.ID] = role
	}
	return repaired
}

func (a *Authorizer) verifyNoCycles() error {
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

func (a *Authorizer) loadUsers(snap *snapshot.Policy) (repaired bool) {
	for _, us := range snap.Users {
		role := a.roles[us.RoleID]
		if role == nil {
			repaired = true
			continue
		}
		user := &model.UserNode{ID: us.ID, Role: role}
		a.users[us.ID] = user
		role.Users[us.ID] = user
	}
	return repaired
}

func (a *Authorizer) loadGrants(snap *snapshot.Policy) (repaired bool) {
	grantSeen := make(map[string]bool)
	for _, gs := range snap.Grants {
		fromRole, toRole := a.roles[gs.FromRoleID], a.roles[gs.ToRoleID]
		if fromRole == nil || toRole == nil {
			repaired = true
			continue
		}
		if gs.FromRoleID == gs.ToRoleID {
			repaired = true
			toRole.GrantedMap[gs.Resource] = true
			continue
		}
		if !toRole.HasAncestor(gs.FromRoleID) {
			repaired = true
			continue
		}
		key := model.GrantKey(gs.FromRoleID, gs.ToRoleID, resource.Resource(gs.Resource))
		if grantSeen[key] {
			repaired = true
			continue
		}
		grantSeen[key] = true
		grant := &model.GrantNode{FromRoleID: gs.FromRoleID, ToRoleID: gs.ToRoleID, Resource: resource.Resource(gs.Resource)}
		fromRole.GrantsOut = append(fromRole.GrantsOut, grant)
		toRole.GrantsIn = append(toRole.GrantsIn, grant)
		toRole.GrantedMap[gs.Resource] = true
	}
	return repaired
}

// save 将当前角色树序列化为快照并全量写入适配器。
func (a *Authorizer) save() error {
	snap := &snapshot.Policy{}
	seen := make(map[string]bool)

	root := a.roles["root"]
	if root == nil {
		return fmt.Errorf("auther: root role not found")
	}

	queue := []*model.RoleNode{root}
	for len(queue) > 0 {
		role := queue[0]
		queue = queue[1:]

		rs := snapshot.Role{ID: role.ID}
		if role.Parent != nil {
			rs.ParentID = role.Parent.ID
		}
		snap.Roles = append(snap.Roles, rs)

		for _, user := range role.Users {
			snap.Users = append(snap.Users, snapshot.User{ID: user.ID, RoleID: user.Role.ID})
		}
		for _, g := range role.GrantsOut {
			key := model.GrantKey(g.FromRoleID, g.ToRoleID, g.Resource)
			if !seen[key] {
				seen[key] = true
				snap.Grants = append(snap.Grants, snapshot.Grant{
					FromRoleID: g.FromRoleID,
					ToRoleID:   g.ToRoleID,
					Resource:   string(g.Resource),
				})
			}
		}
		for _, child := range role.Children {
			queue = append(queue, child)
		}
	}
	return a.adapter.Save(snap)
}
