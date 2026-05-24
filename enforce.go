package auther

import "fmt"

// =============================================================================
// Enforcement API
// =============================================================================

// Enforce checks whether a user has permission to access a resource.
//
// The enforcement model is explicit-only: a user gets access from:
//  1. Their direct role's own Resources
//  2. Grants explicitly given to their role (GrantsIn)
//
// Ancestor role resources and grants do NOT auto-inherit.
func (a *Authorizer) Enforce(userID, res string) (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	user := a.users[userID]
	if user == nil {
		return false, fmt.Errorf("%w: %s", ErrUserNotFound, userID)
	}
	role := user.Role
	if role == nil {
		return false, nil
	}

	for pattern := range role.Resources {
		if match(pattern, res) {
			return true, nil
		}
	}
	for _, g := range role.GrantsIn {
		if match(g.Resource, res) {
			return true, nil
		}
	}
	return false, nil
}

// GetUserPermissions returns all unique resource patterns effective for a user.
func (a *Authorizer) GetUserPermissions(userID string) ([]string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	user := a.users[userID]
	if user == nil {
		return nil, fmt.Errorf("%w: %s", ErrUserNotFound, userID)
	}
	role := user.Role
	if role == nil {
		return nil, nil
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
