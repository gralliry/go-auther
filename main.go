package main

import (
	"github.com/gralliry/go-auther/adapter/empty"
	"github.com/gralliry/go-auther/internal/model"
)

type (
	Resource = model.Resource
	Role     = model.Role
	User     = model.User
	Policy   = model.Policy
)

func main() {
	adapter := empty.New()
	manager, err := model.New(adapter)
	if err != nil {
		panic(err)
	}

	root, exist := manager.GetRole("root")
	if !exist {
		panic("root not exists")
	}

	lzh, err := manager.CreateRole("lzh")
	if err != nil {
		panic(err)
	}

	pl := root.Reviced()

	root.Grant(pl[0], "/a/b/*", lzh)

	println(lzh.Enforce("/a/b/c"))

}
