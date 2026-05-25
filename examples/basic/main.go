// 基本使用示例：演示角色创建、用户管理、资源授权和权限检查。
package main

import (
	"fmt"
	"os"

	"auther"
	"auther/adapters/memory"
)

func main() {
	a, err := auther.NewAuthorizer(memoryadapter.NewMemoryAdapter())
	if err != nil {
		fmt.Fprintln(os.Stderr, "创建 Authorizer 失败:", err)
		os.Exit(1)
	}

	// 创建角色层级：root -> admin -> editor
	must(a.CreateRole("root", "admin"))
	must(a.CreateRole("admin", "editor"))

	// admin 拥有 /user/* 权限
	must(a.Grant("admin", "admin", "/user/*"))
	// editor 拥有 /data/* 权限
	must(a.Grant("editor", "editor", "/data/*"))
	// root 向 admin 授权 /g/**
	must(a.Grant("root", "admin", "/g/**"))

	// 创建用户
	must(a.CreateUser("admin", "u_admin"))
	must(a.CreateUser("editor", "u_editor"))

	// 权限检查
	checks := []struct{ user, resource string }{
		{"u_admin", "/user/create"},
		{"u_admin", "/g/anything"},
		{"u_admin", "/data/read"},
		{"u_editor", "/data/read"},
		{"u_editor", "/user/create"},
	}

	for _, c := range checks {
		ok, _ := a.Enforce(c.user, c.resource)
		fmt.Printf("Enforce(%s, %s) = %v\n", c.user, c.resource, ok)
	}
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
