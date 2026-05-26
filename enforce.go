package auther

import (
	"fmt"
)

// Enforce checks whether a user is authorized to access the given resource.
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
	if user.Role == nil {
		return false, fmt.Errorf("auther: user %s has no role — corrupted state", userID)
	}

	role := user.Role
	return role.HasResource(string(normalized)), nil
}
