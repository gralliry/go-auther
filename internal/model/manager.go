package model

import (
	"errors"

	// var q deque.Deque[*Role]

	"github.com/gralliry/go-auther/adapter"
	"github.com/gralliry/go-auther/internal/pkg/algo"
	"github.com/gralliry/go-auther/internal/pkg/set"
)

type (
	Adapter = adapter.Adapter
)

type Manager struct {
	roles *set.AutoCacheMap[string, *Role]
	users *set.AutoCacheMap[string, *User]

	adapter Adapter
}

func New(adapter Adapter) (*Manager, error) {
	// 参数校验
	if adapter == nil {
		return nil, errors.New("adapter is nil")
	}
	// 初始化adapter
	m := &Manager{
		roles:   set.NewAutoCacheMap[string, *Role](),
		users:   set.NewAutoCacheMap[string, *User](),
		adapter: adapter,
	}

	// load roleInfo | users | grants
	data, err := m.adapter.All()
	if err != nil {
		return nil, err
	}

	// 构建 role
	for _, info := range data.Role {
		role := newRole(info.ID)
		m.roles.Add(role)
	}
	rootRole := newRole("root")
	m.roles.Add(rootRole)

	// 构建 user
	for _, info := range data.User {
		// 构建 user
		user, exist := m.users.Get(info.ID)
		if !exist {
			user = newUser(info.ID)
			m.users.Add(user)
		}
		// 添加 role 到 user
		role, exist := m.roles.Get(info.RoleID)
		if exist {
			user.roles.Add(role)
		}
		// 添加 grant 到 role
		m.users.Add(user)
	}

	rootPolicy := newPolicy(0, Resource("/**"))
	rootRole.srcGrants.Add(rootPolicy)

	// 构建 policy
	policyMap := make(map[int64]int64)
	for _, info := range data.Policy {
		if !m.roles.HasByKey(info.GrantorRoleID) {
			continue
		}
		if !m.roles.HasByKey(info.GranteeRoleID) {
			continue
		}
		policyMap[info.ID] = info.ParentID
	}
	// 剪枝无效的 policy
	policyMap = algo.PruneTree(0, policyMap)
	for _, info := range data.Policy {
		if _, ok := policyMap[info.ID]; !ok {
			continue
		}
		grantor, exist := m.roles.Get(info.GrantorRoleID)
		if !exist {
			continue
		}
		grantee, exist := m.roles.Get(info.GranteeRoleID)
		if !exist {
			continue
		}
		policy := newPolicy(info.ID, Resource(info.Resource))
		// 构建关系链
		grantor.tarGrants.Add(policy)
		grantee.srcGrants.Add(policy)
	}

	return m, nil
}

func (m *Manager) CreateRole(roleID string) (*Role, error) {
	if m.roles.HasByKey(roleID) {
		return nil, errors.New("role already exists")
	}
	role := newRole(roleID)
	m.roles.Add(role)
	return role, nil
}

func (m *Manager) GetRole(roleID string) (*Role, bool) {
	role, exist := m.roles.Get(roleID)
	return role, exist
}

func (m *Manager) CreateUser(userID string) (*User, error) {
	if m.users.HasByKey(userID) {
		return nil, errors.New("user already exists")
	}
	user := newUser(userID)
	m.users.Add(user)
	return user, nil
}

func (m *Manager) GetUser(userID string) (*User, bool) {
	user, exist := m.users.Get(userID)
	return user, exist
}
