package auther

import (
	"fmt"

	"github.com/gralliry/go-auther/internal/model"
)

// Enforce 检查用户是否有权限访问指定资源。
func (a *Authorizer) Enforce(userID, res string) (bool, error) {
	normalized, err := normalizeRes(res)
	if err != nil {
		return false, err
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	role, err := a.lookupUserRole(userID)
	if err != nil {
		return false, err
	}
	return role.HasResource(normalized), nil
}

// Permissions 返回用户当前生效的所有资源权限模式。
func (a *Authorizer) Permissions(userID string) ([]string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role, err := a.lookupUserRole(userID)
	if err != nil {
		return nil, err
	}

	return role.Resources(), nil
}

// lookupUserRole 查找用户并返回其所属角色。
func (a *Authorizer) lookupUserRole(userID string) (*model.RoleNode, error) {
	user := a.users[userID]
	if user == nil {
		return nil, fmt.Errorf("%w: %s", ErrUserNotFound, userID)
	}
	if user.Role == nil {
		return nil, fmt.Errorf("auther: user %s has no role — corrupted state", userID)
	}
	return user.Role, nil
}
