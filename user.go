package auther

import (
	"fmt"

	"github.com/gralliry/go-auther/internal/model"
	"github.com/gralliry/go-auther/snapshot"
)

// UserInfo is the public view of a user entity.
type UserInfo struct {
	ID     string
	RoleID string
}

// CreateUser creates a new user under the specified role.
// Users are passive leaf nodes — they inherit their role's effective
// permissions but cannot manage resources or create other users/roles.
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

	return a.adapter.SetUser(snapshot.User{ID: userID, RoleID: roleID})
}

// DeleteUser removes a user from the specified role.
// roleID must match the user's current role.
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

	return a.adapter.UnsetUser(snapshot.User{ID: userID, RoleID: roleID})
}

// GetUser returns information for the specified user.
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

// GetUsers returns all users in the system.
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
