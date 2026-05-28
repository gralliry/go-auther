package model

import (
	"errors"
	"sync"

	"github.com/gammazero/deque"
	"github.com/gralliry/go-auther/adapter/memory"
	"github.com/gralliry/go-auther/internal/pkg/set"
)

type Option func(*Config)

type Config struct {
	Adapter Adapter
	Fixed   bool
}

func WithAdapter(adapter Adapter) Option {
	return func(cfg *Config) { cfg.Adapter = adapter }
}

func WithFixed(fixed bool) Option {
	return func(cfg *Config) { cfg.Fixed = fixed }
}

type Manager struct {
	root *Role

	namespace set.Set[string]
	roles     set.ValueSet[string, *Role]
	users     set.ValueSet[string, *User]

	mtx     sync.RWMutex
	adapter Adapter
}

func (m *Manager) Init(opts ...Option) error {
	// 初始化配置
	cfg := &Config{
		Adapter: memory.New(),
		Fixed:   true,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	// 参数校验
	if cfg.Adapter == nil {
		return errors.New("adapter is nil")
	}
	// 初始化adapter
	m.adapter = cfg.Adapter
	// 初始化root
	root := rawRole("root")
	// set: root
	root.parent = root
	// load roleInfo | users | grants
	roleInfo, err := m.adapter.AllRoles()
	if err != nil {
		return err
	}
	userInfo, err := m.adapter.AllUsers()
	if err != nil {
		return err
	}
	grantInfo, err := m.adapter.AllGrants()
	if err != nil {
		return err
	}
	_ = grantInfo
	_ = userInfo
	_ = roleInfo
	// 构建role tree
	roleMap := make(map[string]*Role)
	for _, role := range roleInfo {
		childID := role[0]
		roleMap[childID] = rawRole(childID)
	}
	for _, role := range roleInfo {
		childID, parentID := role[0], role[1]
		// role 一定存在
		childNode := roleMap[childID]
		parentNode, ok := roleMap[parentID]
		if !ok {
			continue
		}
		// parent 添加 child
		parentNode.children.Add(childNode)
		// child 设置 parent
		childNode.parent = parentNode
	}
	clear(roleMap)
	clear(roleInfo)
	// 广度优先遍历，构建role tree // roleList 查找有效role
	var q deque.Deque[*Role]
	q.PushBack(root)
	for q.Len() > 0 {
		node := q.PopFront()
		roleMap[node.ID()] = node
		for _, child := range node.children.ToSlice() {
			q.PushBack(child)
		}
	}
	q.Clear()

	// 初始化任务

	return nil
}

func (m *Manager) CreateRole(parent *Role, id string) (*Role, error) {
	if !parent.Valid() {
		return nil, ErrRoleInvalid
	}
	child := rawRole(id)
	// 设置 parent
	child.parent = parent
	// parent 添加 child
	parent.children.Add(child)
	return child, nil
}
