package model

import (
	"github.com/gralliry/go-auther/adapter"
	"github.com/gralliry/go-auther/errors"
	"github.com/gralliry/go-auther/internal/pkg/set"
)

type Manager struct {
	roles *set.AutoCacheMap[string, *Role]
	users *set.AutoCacheMap[string, *User]

	area *Area
}

// New creates a Manager by loading persisted state from the adapter. The root role
// is always present with a /? policy.
func New(adapter Adapter) (*Manager, error) {
	if adapter == nil {
		return nil, errors.ErrAdapterRequired
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

	rootPolicy := newPolicy(0, NewResource("/**"), m.area)
	rootPolicy.parents = 1
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
		policy := newPolicy(info.ID, NewResource(info.Resource), m.area)
		grantor.tarGrants.Add(policy)
		grantee.srcGrants.Add(policy)
	}

	// Rebuild DAG: compute parents count and children links from containment.
	for _, info := range data.Policy {
		grantor, exist := m.roles.Get(info.GrantorRoleID)
		if !exist {
			continue
		}
		grantee, exist := m.roles.Get(info.GranteeRoleID)
		if !exist {
			continue
		}
		// Find this policy by ID from grantee's srcGrants (it was just added).
		var policy *Policy
		grantee.srcGrants.Range(func(p *Policy) {
			if p.id == info.ID {
				policy = p
			}
		})
		if policy == nil {
			continue
		}
		res := NewResource(info.Resource)
		// Count parent policies in grantor's srcGrants that contain this resource.
		grantor.srcGrants.Range(func(parent *Policy) {
			if parent.contains(res) {
				policy.parents++
				parent.children.Add(policy)
			}
		})
	}

	return m, nil
}

// CreateRole creates a new role with the given ID and persists it.
func (m *Manager) CreateRole(roleID string) (*Role, error) {
	m.area.Lock()
	defer m.area.Unlock()

	if m.roles.HasByKey(roleID) {
		return nil, errors.ErrRoleExists
	}
	if err := m.area.CreateRole(adapter.Role{ID: roleID}); err != nil {
		return nil, err
	}
	role := newRole(roleID, m.area)
	m.roles.Add(role)
	return role, nil
}

// GetRole looks up a role by ID. The second return value reports whether the role was found.
func (m *Manager) GetRole(roleID string) (*Role, bool) {
	m.area.RLock()
	defer m.area.RUnlock()

	role, exist := m.roles.Get(roleID)
	return role, exist
}

// CreateUser creates a new user with the given ID and persists it.
func (m *Manager) CreateUser(userID string) (*User, error) {
	m.area.Lock()
	defer m.area.Unlock()

	if m.users.HasByKey(userID) {
		return nil, errors.ErrUserExists
	}
	if err := m.area.CreateUser(adapter.User{ID: userID}); err != nil {
		return nil, err
	}
	user := newUser(userID, m.area)
	m.users.Add(user)
	return user, nil
}

// GetUser looks up a user by ID. The second return value reports whether the user was found.
func (m *Manager) GetUser(userID string) (*User, bool) {
	m.area.RLock()
	defer m.area.RUnlock()

	user, exist := m.users.Get(userID)
	return user, exist
}
