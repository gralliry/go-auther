package auther

import (
	"fmt"

	"auther/internal/model"
)

// UserInfo 是对外暴露的用户信息视图。
type UserInfo struct {
	ID     string
	RoleID string
}

// CreateUser 在指定角色下创建一个新用户。
// 用户是被动叶子节点 —— 继承所属角色的权限，但不能管理资源或创建其他用户/角色。
func (a *Authorizer) CreateUser(roleID, userID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.users[userID]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateUser, userID)
	}

	role := a.roles[roleID]
	if role == nil {
		return fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}

	user := &model.UserNode{
		ID:   userID,
		Role: role,
	}

	a.users[userID] = user
	role.Users[userID] = user

	return a.save()
}

// DeleteUser 从系统中删除指定用户。
func (a *Authorizer) DeleteUser(userID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	user := a.users[userID]
	if user == nil {
		return fmt.Errorf("%w: %s", ErrUserNotFound, userID)
	}

	if user.Role != nil {
		delete(user.Role.Users, userID)
	}
	delete(a.users, userID)

	return a.save()
}

// GetUser 返回指定用户的详细信息。
func (a *Authorizer) GetUser(userID string) (*UserInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	user := a.users[userID]
	if user == nil {
		return nil, fmt.Errorf("%w: %s", ErrUserNotFound, userID)
	}

	roleID := ""
	if user.Role != nil {
		roleID = user.Role.ID
	}
	return &UserInfo{
		ID:     user.ID,
		RoleID: roleID,
	}, nil
}

// Users 返回系统中所有用户的列表。
func (a *Authorizer) Users() []*UserInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]*UserInfo, 0, len(a.users))
	for _, user := range a.users {
		roleID := ""
		if user.Role != nil {
			roleID = user.Role.ID
		}
		result = append(result, &UserInfo{
			ID:     user.ID,
			RoleID: roleID,
		})
	}
	return result
}
