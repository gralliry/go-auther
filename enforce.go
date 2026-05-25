package auther

import (
	"fmt"

	"auther/match"
)

// Enforce 检查用户是否有权限访问指定资源。
//
// 权限模型为显式授权，用户仅从以下来源获得权限：
//  1. 其所属角色自身拥有的资源（Role.Resources）
//  2. 其所属角色接收到的显式授权（GrantsIn）
//
// 祖先角色的资源和授权不会自动继承。
func (a *Authorizer) Enforce(userID, res string) (bool, error) {
	normalized, err := normalizeResource(res)
	if err != nil {
		return false, err
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	user := a.users[userID]
	if user == nil {
		return false, fmt.Errorf("%w: %s", ErrUserNotFound, userID)
	}
	role := user.Role
	if role == nil {
		return false, fmt.Errorf("auther: user %s has no role — corrupted state", userID)
	}

	// 先遍历角色自有资源，再遍历收到的授权。
	for pattern := range role.Resources {
		if match.Match(pattern, normalized) {
			return true, nil
		}
	}
	for _, g := range role.GrantsIn {
		if match.Match(g.Resource, normalized) {
			return true, nil
		}
	}
	return false, nil
}

// GetUserPermissions 返回用户当前生效的所有资源权限模式（去重）。
func (a *Authorizer) GetUserPermissions(userID string) ([]string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	user := a.users[userID]
	if user == nil {
		return nil, fmt.Errorf("%w: %s", ErrUserNotFound, userID)
	}
	role := user.Role
	if role == nil {
		return nil, fmt.Errorf("auther: user %s has no role — corrupted state", userID)
	}

	seen := make(map[string]bool)
	var result []string
	for pattern := range role.Resources {
		if !seen[pattern] {
			seen[pattern] = true
			result = append(result, pattern)
		}
	}
	for _, g := range role.GrantsIn {
		if !seen[g.Resource] {
			seen[g.Resource] = true
			result = append(result, g.Resource)
		}
	}
	return result, nil
}
