package auther

import (
	"github.com/gralliry/go-auther/adapter"
	"github.com/gralliry/go-auther/internal/model"
)

type (
	Resource = model.Resource
	Role     = model.Role
	User     = model.User
	Policy   = model.Policy
	Manager  = model.Manager
	Adapter  = adapter.Adapter
)

var (
	NewManager  = model.New
	NewResource = model.NewResource
)
