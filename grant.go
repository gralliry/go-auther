package auther

import (
	"fmt"

	"github.com/gralliry/go-auther/internal/model"
	"github.com/gralliry/go-auther/snapshot"
)

// Grant 将资源从祖先角色授权给后代角色。
//
// 授权方必须是接收方的祖先角色（不允许自授权），否则返回 ErrNotAncestor。
func (a *Authorizer) Grant(fromRoleID, toRoleID, resource string) error {
	res, err := normalizeRes(resource)
	if err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if fromRoleID == toRoleID {
		return fmt.Errorf("%w: self-grant is not allowed", ErrNotAncestor)
	}
	fromRole := a.roles[fromRoleID]
	if fromRole == nil {
		return fmt.Errorf("%w: %s", ErrRoleNotFound, fromRoleID)
	}
	toRole := a.roles[toRoleID]
	if toRole == nil {
		return fmt.Errorf("%w: %s", ErrRoleNotFound, toRoleID)
	}
	if !toRole.HasAncestor(fromRoleID) {
		return fmt.Errorf("%w: %s is not an ancestor of %s", ErrNotAncestor, fromRoleID, toRoleID)
	}

	for _, g := range fromRole.GrantsOut {
		if g.ToRoleID == toRoleID && g.Resource == res {
			return fmt.Errorf("%w: %s -> %s %s", ErrDuplicateGrant, fromRoleID, toRoleID, res)
		}
	}

	grant := &model.GrantNode{FromRoleID: fromRoleID, ToRoleID: toRoleID, Resource: res}
	fromRole.GrantsOut = append(fromRole.GrantsOut, grant)
	toRole.GrantsIn = append(toRole.GrantsIn, grant)
	toRole.GrantedMap[res] = true
	toRole.ResetMatchCache()
	return a.adapter.SetGrant(snapshot.Grant{FromRoleID: fromRoleID, ToRoleID: toRoleID, Resource: res})
}

// Revoke 撤销一条授权，并级联清理子树中失效的子授权。
func (a *Authorizer) Revoke(fromRoleID, toRoleID, resource string) error {
	res, err := normalizeRes(resource)
	if err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	if fromRoleID == toRoleID {
		return fmt.Errorf("%w: self-revoke is not allowed", ErrGrantNotFound)
	}
	fromRole := a.roles[fromRoleID]
	toRole := a.roles[toRoleID]
	if fromRole == nil || toRole == nil {
		return fmt.Errorf("%w", ErrGrantNotFound)
	}
	return a.revokeDelegatedLocked(fromRole, toRole, res)
}

// revokeDelegatedLocked 撤销委托授权，并级联清理子树中失效的子授权。
func (a *Authorizer) revokeDelegatedLocked(fromRole, toRole *model.RoleNode, resource string) error {
	found := false
	for i, g := range fromRole.GrantsOut {
		if g.ToRoleID == toRole.ID && g.Resource == resource {
			fromRole.GrantsOut = append(fromRole.GrantsOut[:i], fromRole.GrantsOut[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("%w: %s -> %s %s", ErrGrantNotFound, fromRole.ID, toRole.ID, resource)
	}

	for i, g := range toRole.GrantsIn {
		if g.FromRoleID == fromRole.ID && g.Resource == resource {
			toRole.GrantsIn = append(toRole.GrantsIn[:i], toRole.GrantsIn[i+1:]...)
			break
		}
	}
	if !model.HasGrant(toRole.GrantsIn, resource) {
		delete(toRole.GrantedMap, resource)
	}
	toRole.ResetMatchCache()

	// 级联清理：子树角色的 GrantedMap 已被更新，若不再覆盖某条转授资源则移除该转授。
	for _, r := range toRole.Subtree() {
		r.GrantsOut = model.RemoveGrantsAndCleanup(r.GrantsOut, a.roles)
		r.ResetMatchCache()
	}
	return a.save()
}

// GetGrantsTo 返回指定角色接收到的授权记录。
func (a *Authorizer) GetGrantsTo(roleID string) ([]*model.GrantNode, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}
	return append([]*model.GrantNode(nil), role.GrantsIn...), nil
}

// GetGrantsFrom 返回指定角色发出的授权记录。
func (a *Authorizer) GetGrantsFrom(roleID string) ([]*model.GrantNode, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}
	return append([]*model.GrantNode(nil), role.GrantsOut...), nil
}
