package manager

import (
	"sync"
	"sync/atomic"

	"github.com/bwmarrin/snowflake"
	"github.com/gralliry/go-auther/adapter"
	"github.com/gralliry/go-auther/errors"
	"github.com/gralliry/go-auther/internal/resource"
)

// gid is an atomic counter used to assign unique node IDs to snowflake generators.
// Each Manager instance gets its own snowflake node.
var gid atomic.Int64

// Manager is the top-level orchestrator. All authorization operations go through it.
type Manager struct {
	roles map[string]*Role          // roleID → role
	users map[string]map[string]struct{} // userID → set of roleIDs

	mutex   sync.RWMutex
	adapter adapter.Adapter
	node    *snowflake.Node // unique ID generator for policies
}

// New creates a Manager by loading persisted state from the adapter.
// The root role is always present with a /** policy.
func New(adapter adapter.Adapter) (*Manager, error) {
	if adapter == nil {
		return nil, errors.ErrAdapterRequired
	}
	node, err := snowflake.NewNode(gid.Add(1))
	if err != nil {
		return nil, err
	}
	m := &Manager{
		roles:   make(map[string]*Role),
		users:   make(map[string]map[string]struct{}),
		adapter: adapter,
		node:    node,
	}
	if err := m.load(); err != nil {
		return nil, err
	}
	return m, nil
}

// generateID returns a globally unique snowflake ID for a new policy.
func (m *Manager) generateID() int64 {
	return m.node.Generate().Int64()
}

// load populates the in-memory state from the adapter.
func (m *Manager) load() error {
	data, err := m.adapter.Snapshot()
	if err != nil {
		return err
	}

	// Build roles.
	for _, info := range data.Role {
		role := newRole()
		m.roles[info.ID] = role
	}
	// Ensure root role exists.
	if _, ok := m.roles["root"]; !ok {
		rootRole := newRole()
		m.roles["root"] = rootRole
	}

	// Build users: create them and assign their roles.
	for _, info := range data.User {
		if _, ok := m.users[info.ID]; !ok {
			m.users[info.ID] = make(map[string]struct{})
		}
	}
	for _, info := range data.User {
		userRoles, ok := m.users[info.ID]
		if !ok {
			continue
		}
		if _, ok := m.roles[info.RoleID]; ok {
			userRoles[info.RoleID] = struct{}{}
		}
	}

	// Ensure root has the /** policy.
	rootRole, _ := m.roles["root"]
	if rootRole != nil {
		hasRootPolicy := false
		rootRole.srcGrants.Range(func(p *Policy) {
			if p.id == 0 {
				hasRootPolicy = true
			}
		})
		if !hasRootPolicy {
			rootPolicy := newPolicy(0, resource.NewResource("/**"))
			rootPolicy.parents = 1
			rootRole.srcGrants.Add(rootPolicy)
		}
	}

	// Build policies: create and link them.
	type policyEdge struct {
		grantorID string
		granteeID string
	}
	policyEdges := make(map[int64]policyEdge, len(data.Policy))

	for _, info := range data.Policy {
		grantor, ok := m.roles[info.GrantorRoleID]
		if !ok {
			continue
		}
		grantee, ok := m.roles[info.GranteeRoleID]
		if !ok {
			continue
		}
		pol := newPolicy(info.ID, resource.NewResource(info.Resource))
		grantor.tarGrants.Add(pol)
		grantee.srcGrants.Add(pol)
		policyEdges[info.ID] = policyEdge{grantorID: info.GrantorRoleID, granteeID: info.GranteeRoleID}
	}

	// Rebuild DAG edges: compute parent counters and children links.
	for _, info := range data.Policy {
		edge, ok := policyEdges[info.ID]
		if !ok {
			continue
		}
		grantor, _ := m.roles[edge.grantorID]
		grantee, _ := m.roles[edge.granteeID]
		if grantor == nil || grantee == nil {
			continue
		}
		var pol *Policy
		grantee.srcGrants.Range(func(p *Policy) {
			if p.id == info.ID {
				pol = p
			}
		})
		if pol == nil {
			continue
		}
		res := resource.NewResource(info.Resource)
		grantor.srcGrants.Range(func(parent *Policy) {
			if parent.contains(res) {
				pol.parents++
				parent.children.Add(pol)
			}
		})
	}

	return nil
}
