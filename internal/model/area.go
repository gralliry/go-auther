package model

import (
	"sync"

	"github.com/bwmarrin/snowflake"
	"github.com/gralliry/go-auther/adapter"
)

type (
	Adapter = adapter.Adapter
)

type Area struct {
	sync.RWMutex
	Adapter
	node *snowflake.Node
}

// NewArea creates an Area initialized with the given adapter and a snowflake node.
func NewArea(adapter Adapter) (*Area, error) {
	node, err := snowflake.NewNode(1)
	if err != nil {
		return nil, err
	}
	return &Area{
		Adapter: adapter,
		node:    node,
	}, nil
}

// GenerateID returns a globally unique snowflake ID for new policies.
func (a *Area) GenerateID() int64 {
	return a.node.Generate().Int64()
}
