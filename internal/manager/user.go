package manager

import (
	"github.com/gralliry/go-auther/adapter"
	"github.com/gralliry/go-auther/errors"
)

// CreateUser creates a new user in memory only. The user is not persisted to
// the adapter until their first role is assigned via Assign — the adapter
// contract requires every user to have at least one role.
func (m *Manager) CreateUser(userID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.users[userID]; ok {
		return errors.ErrUserExists
	}
	m.users[userID] = make(map[string]struct{})
	return nil
}

// CheckUser reports whether the user exists.
func (m *Manager) CheckUser(userID string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	_, ok := m.users[userID]
	return ok
}

// DeleteUser removes the user and persists the deletion.
func (m *Manager) DeleteUser(userID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.users[userID]; !ok {
		return errors.ErrUserNotFound
	}

	if err := m.adapter.DeleteUser(adapter.User{ID: userID}); err != nil {
		return err
	}

	delete(m.users, userID)
	return nil
}

// Assign adds a role to a user.
func (m *Manager) Assign(userID, roleID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	userRoles, ok := m.users[userID]
	if !ok {
		return errors.ErrUserNotFound
	}
	if _, ok := m.roles[roleID]; !ok {
		return errors.ErrRoleNotFound
	}
	if _, ok := userRoles[roleID]; ok {
		return errors.ErrRoleAlreadyAssigned
	}
	if err := m.adapter.LinkUser(adapter.User{ID: userID, RoleID: roleID}); err != nil {
		return err
	}
	userRoles[roleID] = struct{}{}
	return nil
}

// Unassign removes a role from a user and persists the removal.
func (m *Manager) Unassign(userID, roleID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	userRoles, ok := m.users[userID]
	if !ok {
		return errors.ErrUserNotFound
	}
	if _, ok := m.roles[roleID]; !ok {
		return errors.ErrRoleNotFound
	}
	if _, ok := userRoles[roleID]; !ok {
		return errors.ErrRoleNotAssigned
	}
	if err := m.adapter.UnlinkUser(adapter.User{ID: userID, RoleID: roleID}); err != nil {
		return err
	}
	delete(userRoles, roleID)
	return nil
}

// IsAssigned reports whether the user currently has the given role.
func (m *Manager) IsAssigned(userID, roleID string) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	userRoles, ok := m.users[userID]
	if !ok {
		return false, errors.ErrUserNotFound
	}
	if _, ok := m.roles[roleID]; !ok {
		return false, errors.ErrRoleNotFound
	}
	_, ok = userRoles[roleID]
	return ok, nil
}

// EnforceByUser checks whether a user has access to a resource through any
// assigned role.
func (m *Manager) EnforceByUser(userID, res string) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	userRoles, ok := m.users[userID]
	if !ok {
		return false, errors.ErrUserNotFound
	}
	for roleID := range userRoles {
		role, ok := m.roles[roleID]
		if !ok {
			delete(userRoles, roleID)
			continue
		}
		if role.enforce(res) {
			return true, nil
		}
	}
	return false, nil
}
