package auther

import "fmt"

// =============================================================================
// User API
// =============================================================================

// CreateUser creates a new user under the given role.
// Users are passive leaves — they inherit the role's permissions but
// cannot manage resources or create other users/roles.
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

	user := &UserNode{
		ID:   userID,
		Role: role,
	}

	a.users[userID] = user
	role.Users[userID] = user

	return a.save()
}

// DeleteUser removes a user from the system.
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

// GetUser returns information about a user.
func (a *Authorizer) GetUser(userID string) (*UserInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	user := a.users[userID]
	if user == nil {
		return nil, fmt.Errorf("%w: %s", ErrUserNotFound, userID)
	}

	return &UserInfo{
		ID:     user.ID,
		RoleID: user.Role.ID,
	}, nil
}

// GetAllUsers returns all users in the system.
func (a *Authorizer) GetAllUsers() []*UserInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]*UserInfo, 0, len(a.users))
	for _, user := range a.users {
		result = append(result, &UserInfo{
			ID:     user.ID,
			RoleID: user.Role.ID,
		})
	}
	return result
}
