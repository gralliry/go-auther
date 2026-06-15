package auther

import (
	"github.com/gralliry/go-auther/adapter"
	"github.com/gralliry/go-auther/internal/manager"
)

type (
	Role    = manager.Role
	Manager = manager.Manager
	Adapter = adapter.Adapter
)

var (
	NewManager = manager.New
)
