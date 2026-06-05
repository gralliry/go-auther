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

func (a *Area) GenerateID() int64 {
	return a.node.Generate().Int64()
}
