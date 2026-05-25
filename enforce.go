package auther

import (
	"fmt"

	"auther/internal/match"
)

// Enforce 检查用户是否有权限访问指定资源。
//
// 权限模型为显式授权，权限仅来自角色接收到的授权记录（GrantsIn）。
// 祖先角色的资源和授权不会自动继承。
func (a *Authorizer) Enforce(userID, res string) (bool, error) {
	normalized, err := normalizeRes(res)
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

	// 快速路径：精确资源匹配 O(1)
	if role.GrantedMap[normalized] {
		return true, nil
	}

	// 匹配缓存：避免重复的 glob 匹配开销
	if cached, ok := role.GetMatchCache(normalized); ok {
		return cached, nil
	}

	for _, g := range role.GrantsIn {
		if match.Match(g.Resource, normalized) {
			role.SetMatchCache(normalized, true)
			return true, nil
		}
	}

	role.SetMatchCache(normalized, false)
	return false, nil
}

// Permissions 返回用户当前生效的所有资源权限模式（去重）。
func (a *Authorizer) Permissions(userID string) ([]string, error) {
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
	for _, g := range role.GrantsIn {
		if !seen[g.Resource] {
			seen[g.Resource] = true
			result = append(result, g.Resource)
		}
	}
	return result, nil
}
