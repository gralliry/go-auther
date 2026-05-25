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

	return a.saveSetUser(roleID, userID)
}

// DeleteUser 从指定角色中删除用户。roleID 必须与用户所属角色匹配。
func (a *Authorizer) DeleteUser(roleID, userID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	role := a.roles[roleID]
	if role == nil {
		return fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}

	user := a.users[userID]
	if user == nil {
		return fmt.Errorf("%w: %s", ErrUserNotFound, userID)
	}

	if user.Role == nil || user.Role.ID != roleID {
		return fmt.Errorf("%w: user %s does not belong to role %s", ErrUserNotFound, userID, roleID)
	}

	delete(role.Users, userID)
	delete(a.users, userID)

	return a.saveUnsetUser(roleID, userID)
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

// GetUsers 返回系统中所有用户的列表。
func (a *Authorizer) GetUsers() []*UserInfo {
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
