package model

import (
	"errors"

	"github.com/gralliry/go-auther/adapter"
	"github.com/gralliry/go-auther/internal/pkg/set"
)

type (
	Adapter = adapter.Adapter
)

type Manager struct {
	roles *set.AutoCacheMap[string, *Role]
	users *set.AutoCacheMap[string, *User]

	area *Area
}

func New(adapter Adapter) (*Manager, error) {
	if adapter == nil {
		return nil, errors.New("adapter is nil")
	}
	area, err := NewArea(adapter)
	if err != nil {
		return nil, err
	}
	m := &Manager{
		roles: set.NewAutoCacheMap[string, *Role](),
		users: set.NewAutoCacheMap[string, *User](),
		area:  area,
	}

	data, err := m.area.All()
	if err != nil {
		return nil, err
	}

	// Build roles.
	for _, info := range data.Role {
		role := newRole(info.ID, m.area)
		m.roles.Add(role)
	}
	rootRole := newRole("root", m.area)
	m.roles.Add(rootRole)

	// Build users.
	for _, info := range data.User {
		user, exist := m.users.Get(info.ID)
		if !exist {
			user = newUser(info.ID, m.area)
			m.users.Add(user)
		}
		role, exist := m.roles.Get(info.RoleID)
		if exist {
			user.roles.Add(role)
		}
		m.users.Add(user)
	}

	rootPolicy := newPolicy(0, Resource("/**"), m.area)
	rootRole.srcGrants.Add(rootPolicy)

	// Build policies.
	for _, info := range data.Policy {
		grantor, exist := m.roles.Get(info.GrantorRoleID)
		if !exist {
			continue
		}
		grantee, exist := m.roles.Get(info.GranteeRoleID)
		if !exist {
			continue
		}
		policy := newPolicy(info.ID, Resource(info.Resource), m.area)
		grantor.tarGrants.Add(policy)
		grantee.srcGrants.Add(policy)
	}

	return m, nil
}

func (m *Manager) CreateRole(roleID string) (*Role, error) {
	m.area.Lock()
	defer m.area.Unlock()

	if m.roles.HasByKey(roleID) {
		return nil, errors.New("role already exists")
	}
	if err := m.area.CreateRole(adapter.Role{ID: roleID}); err != nil {
		return nil, err
	}
	role := newRole(roleID, m.area)
	m.roles.Add(role)
	return role, nil
}

func (m *Manager) GetRole(roleID string) (*Role, bool) {
	m.area.RLock()
	defer m.area.RUnlock()

	role, exist := m.roles.Get(roleID)
	return role, exist
}

func (m *Manager) CreateUser(userID string) (*User, error) {
	m.area.Lock()
	defer m.area.Unlock()

	if m.users.HasByKey(userID) {
		return nil, errors.New("user already exists")
	}
	if err := m.area.CreateUser(adapter.User{ID: userID}); err != nil {
		return nil, err
	}
	user := newUser(userID, m.area)
	m.users.Add(user)
	return user, nil
}

func (m *Manager) GetUser(userID string) (*User, bool) {
	m.area.RLock()
	defer m.area.RUnlock()

	user, exist := m.users.Get(userID)
	return user, exist
}
