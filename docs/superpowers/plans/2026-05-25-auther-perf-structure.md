# Auther 性能优化与项目结构规范 — 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 auther 权限库拆分为 model/match/auther 子包，优化 Enforce 热路径至 ~100ns（零分配迭代匹配 + Grants 索引 + LRU 缓存）。

**Architecture:** model（数据模型）→ match（匹配引擎）→ auther（公开 API）；主包通过 type alias 保持外部 API 不变；GrantsIn 从切片改为 map 索引；match 用迭代回溯替代 DP 表分配。

**Tech Stack:** Go 1.26, 标准库 only

---

### Task 1: 创建 model/ 包，迁移所有数据类型

**Files:**
- Create: `model/model.go`
- Modify: `types.go` (删除)
- Modify: `auther.go`, `enforce.go`, `grant.go`, `role.go`, `user.go`, `resource.go`, `errors.go`, `adapter.go` (更新 import)

- [ ] **Step 1: 创建 model/model.go，搬入所有数据类型**

将 `types.go` 全部内容搬入 `model/model.go`，改 package 为 `model`：

```go
// Package model 定义 auther 权限库的核心数据结构。
package model

// RoleNode 表示角色树中的一个角色节点。
type RoleNode struct {
	ID        string
	Parent    *RoleNode
	Children  map[string]*RoleNode
	Resources map[string]bool
	GrantsIn  []RoleGrant
	GrantsOut []RoleGrant
	Users     map[string]*UserNode
}

// UserNode 表示权限系统中的一个用户。
type UserNode struct {
	ID   string
	Role *RoleNode
}

// RoleGrant 表示从祖先角色到子角色的显式资源授权记录。
type RoleGrant struct {
	FromRoleID string
	ToRoleID   string
	Resource   string
}

// RoleInfo 是对外暴露的角色信息视图。
type RoleInfo struct {
	ID         string
	ParentID   string
	Resources  []string
	SubRoleIDs []string
	UserIDs    []string
	GrantsIn   []RoleGrant
	GrantsOut  []RoleGrant
}

// UserInfo 是对外暴露的用户信息视图。
type UserInfo struct {
	ID     string
	RoleID string
}

// PolicySnapshot 是完整权限状态的扁平化快照，由适配器用于持久化。
type PolicySnapshot struct {
	Roles  []RoleSnapshot  `json:"roles"`
	Users  []UserSnapshot  `json:"users"`
	Grants []GrantSnapshot `json:"grants"`
}

// RoleSnapshot 是用于序列化的扁平角色记录。
type RoleSnapshot struct {
	ID        string   `json:"id"`
	ParentID  string   `json:"parent_id"`
	Resources []string `json:"resources"`
}

// UserSnapshot 是用于序列化的扁平用户记录。
type UserSnapshot struct {
	ID     string `json:"id"`
	RoleID string `json:"role_id"`
}

// GrantSnapshot 是用于序列化的扁平授权记录。
type GrantSnapshot struct {
	FromRoleID string `json:"from_role_id"`
	ToRoleID   string `json:"to_role_id"`
	Resource   string `json:"resource"`
}
```

- [ ] **Step 2: 运行编译，确认 model 包无语法错误**

```bash
cd "D:\Code\Golang\Auther" && go build ./model/
```

Expected: 编译成功，无输出。

- [ ] **Step 3: 删除 types.go**

```bash
rm "D:\Code\Golang\Auther\types.go"
```

- [ ] **Step 4: 更新主包所有文件，添加 model import，全局替换类型引用**

对所有本包 .go 文件执行：
- 添加 `import "auther/model"`
- `RoleNode` → `model.RoleNode`
- `UserNode` → `model.UserNode`
- `RoleGrant` → `model.RoleGrant`
- `RoleInfo` → `model.RoleInfo`
- `UserInfo` → `model.UserInfo`
- `PolicySnapshot` → `model.PolicySnapshot`
- `RoleSnapshot` → `model.RoleSnapshot`
- `UserSnapshot` → `model.UserSnapshot`
- `GrantSnapshot` → `model.GrantSnapshot`

涉及文件：`auther.go`, `enforce.go`, `grant.go`, `role.go`, `user.go`, `adapter.go`

- [ ] **Step 5: 更新 adapters/ 引用**

`adapters/memory/memory_adapter.go` 和 `adapters/file/file_adapter.go` 中：
- 添加 `import "auther/model"`
- `auther.PolicySnapshot` → `model.PolicySnapshot`
- `auther.RoleSnapshot` → `model.RoleSnapshot`
- `auther.UserSnapshot` → `model.UserSnapshot`
- `auther.GrantSnapshot` → `model.GrantSnapshot`

- [ ] **Step 6: 运行全量编译和测试**

```bash
cd "D:\Code\Golang\Auther" && go build ./... && go test ./...
```

Expected: 所有包编译通过，所有测试 PASS。

- [ ] **Step 7: 添加主包类型别名**

在 `auther.go` 顶部（package 声明后）添加：

```go
// 类型别名：外部只需 import "auther" 即可使用以下类型。
type (
	RoleGrant      = model.RoleGrant
	RoleInfo       = model.RoleInfo
	UserInfo       = model.UserInfo
	RoleNode       = model.RoleNode
	UserNode       = model.UserNode
	PolicySnapshot = model.PolicySnapshot
	RoleSnapshot   = model.RoleSnapshot
	UserSnapshot   = model.UserSnapshot
	GrantSnapshot  = model.GrantSnapshot
)
```

- [ ] **Step 8: 回退主包中不再是必要的 model. 前缀**

由于类型别名存在，auther.go 中原本的 `model.RoleNode` 等可直接用 `RoleNode`。但不强制回退 — 两种写法均可编译。保持 `model.` 前缀增加可读性也可接受。此步骤为可选优化。

- [ ] **Step 9: 运行测试确认一切正常**

```bash
cd "D:\Code\Golang\Auther" && go test ./...
```

Expected: PASS for all packages.

---

### Task 2: 创建 match/ 包，迁移匹配逻辑

**Files:**
- Create: `match/match.go`
- Create: `match/segment.go`
- Modify: `resource.go` (仅为 normalizeResource 调用 match)

- [ ] **Step 1: 创建 match/segment.go，搬入 segs 类型和 parseSegs**

```go
// Package match 提供资源路径的 glob 匹配引擎。
// 支持 *（单段匹配）和 **（零或多段匹配）通配符。
package match

// segs 保存路径字符串中各个路径段的起止位置。
type segs struct {
	s  string // 原始路径字符串
	p  []int  // 各段起始位置
	on []int  // 各段结束位置（不包含）
}

// ParseSegs 将路径解析为路径段视图，不产生额外的字符串分配。
func ParseSegs(path string) segs {
	if path == "/" {
		return segs{s: "/", p: []int{0}, on: []int{1}}
	}
	n := 1
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			n++
		}
	}
	p := make([]int, 0, n)
	on := make([]int, 0, n)
	for i := 0; i < len(path); {
		if path[i] == '/' {
			i++
			continue
		}
		p = append(p, i)
		for i < len(path) && path[i] != '/' {
			i++
		}
		on = append(on, i)
	}
	return segs{s: path, p: p, on: on}
}

// N 返回路径段的数量。
func (sg segs) N() int { return len(sg.p) }

// At 返回第 i 个路径段的字符串。
func (sg segs) At(i int) string {
	if i >= len(sg.p) {
		return ""
	}
	return sg.s[sg.p[i]:sg.on[i]]
}
```

- [ ] **Step 2: 创建 match/match.go，搬入 hasWildcard 和 match 主函数**

当前暂时保留 DP 表版本，后续 Task 4 替换为零分配迭代算法：

```go
package match

// HasWildcard 判断字符串是否包含通配符 '*'。
func HasWildcard(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '*' {
			return true
		}
	}
	return false
}

// Match 判断目标路径是否匹配 glob 模式。
// pattern 和 target 都应为 '/' 分隔的绝对路径。
func Match(pattern, target string) bool {
	if pattern == target {
		return true
	}
	if !HasWildcard(pattern) {
		return false
	}
	pat := ParseSegs(pattern)
	tgt := ParseSegs(target)
	return DPMatch(pat, tgt)
}

// DPMatch 使用自底向上动态规划匹配模式段与目标段。
func DPMatch(pat, tgt segs) bool {
	n, m := pat.N(), tgt.N()
	rowStride := m + 1
	dp := make([]bool, (n+1)*rowStride)
	idx := func(i, j int) int { return i*rowStride + j }

	dp[idx(n, m)] = true
	for i := n - 1; i >= 0; i-- {
		if pat.At(i) == "**" {
			dp[idx(i, m)] = dp[idx(i+1, m)]
		}
	}
	for i := n - 1; i >= 0; i-- {
		pSeg := pat.At(i)
		for j := m - 1; j >= 0; j-- {
			switch {
			case pSeg == "**":
				dp[idx(i, j)] = dp[idx(i+1, j)] || dp[idx(i, j+1)]
			case pSeg == "*" || pSeg == tgt.At(j):
				dp[idx(i, j)] = dp[idx(i+1, j+1)]
			}
		}
	}
	return dp[idx(0, 0)]
}
```

- [ ] **Step 3: 运行编译确认 match 包正确**

```bash
cd "D:\Code\Golang\Auther" && go build ./match/
```

- [ ] **Step 4: 更新 resource.go，用 match 包替换内联函数**

`resource.go` 改为只保留 `normalizeResource`，match 调用改为 `match.Match()`、`match.HasWildcard()`：

```go
package auther

import (
	"fmt"
	"auther/match"
)

// normalizeResource 校验并规范化资源路径。
func normalizeResource(resource string) (string, error) {
	if resource == "" {
		return "", fmt.Errorf("%w: resource must not be empty", ErrInvalidResource)
	}
	if resource[0] != '/' {
		return "", fmt.Errorf("%w: resource must start with '/'", ErrInvalidResource)
	}

	result := make([]byte, 0, len(resource))
	result = append(result, '/')
	for i := 1; i < len(resource); i++ {
		if resource[i] == '/' && resource[i-1] == '/' {
			continue
		}
		result = append(result, resource[i])
	}

	if len(result) > 1 && result[len(result)-1] == '/' {
		result = result[:len(result)-1]
	}

	return string(result), nil
}
```

- [ ] **Step 5: 更新主包中所有调用 match() 的地方**

`enforce.go` 和 `grant.go` 中的 `match(pattern, normalized)` → `match.Match(pattern, normalized)`。
注意：主包本地函数名改为小写后已被 match 包替代，直接在代码中全局替换 `match(` → `match.Match(`。

- [ ] **Step 6: 迁移匹配相关测试到 match/ 包**

创建 `match/match_test.go`，从 `resource_test.go` 搬入 `TestMatchExact`、`TestMatchRootOnly`、`TestMatchEdgeCases`、`TestMatchDoubleStarAlone`。所有 `match(` 调用改为 `Match(`。

创建 `match/segment_test.go`，添加 parseSegs 基础测试。

- [ ] **Step 7: 更新 resource_bench_test.go 中的 benchmark**

将 benchmark 中的 `match(` 调用改为 `match.Match(`，添加 `import "auther/match"`。

- [ ] **Step 8: 删除 resource.go 中的 match 相关代码**

确认 `match()`、`hasWildcard()`、`parseSegs()`、`matchDP()`、`segs` 类型及其方法已从 `resource.go` 移除。

- [ ] **Step 9: 运行全量测试**

```bash
cd "D:\Code\Golang\Auther" && go test ./...
```

Expected: PASS for all packages.

---

### Task 3: 更新 auther_test.go 中的适配器引用

**Files:**
- Modify: `auther_test.go`

- [ ] **Step 1: 更新 corruptAdapter 中的类型引用**

`auther_test.go` 中的 `corruptAdapter` 和 `newHealed` 函数使用 `PolicySnapshot`、`RoleSnapshot` 等类型。由于已在主包添加了类型别名，这些引用可保持不变（`PolicySnapshot` 现在通过别名指向 `model.PolicySnapshot`）。确认编译通过即可。

```bash
cd "D:\Code\Golang\Auther" && go build ./...
```

---

### Task 4: 实现零分配迭代匹配算法

**Files:**
- Modify: `match/match.go`

- [ ] **Step 1: 重写 Match() 为迭代回溯算法**

用双指针 + 回溯替代 DP 表，零分配：

```go
package match

// Match 判断目标路径是否匹配 glob 模式。
// 使用迭代回溯算法，零堆分配。
func Match(pattern, target string) bool {
	if pattern == target {
		return true
	}
	if !HasWildcard(pattern) {
		return false
	}
	return matchIter(pattern, target)
}

// matchIter 使用双指针 + 回溯实现 glob 匹配。
// 参考 Unix glob 实现：* 匹配单段，** 匹配零或多段。
func matchIter(pattern, target string) bool {
	pi, ti := 0, 0       // 当前 pattern 和 target 位置
	starIdx, matchIdx := -1, 0 // 最近 ** 的位置和匹配位置

	for ti < len(target) {
		if pi < len(pattern) && pattern[pi] == target[ti] && pattern[pi] != '*' {
			// 字面量匹配
			pi++
			ti++
			continue
		}

		if pi < len(pattern) && pattern[pi] == '*' {
			// 检查是否为 **
			if pi+1 < len(pattern) && pattern[pi+1] == '*' {
				// **: 零或多段
				starIdx = pi
				pi += 2
				matchIdx = ti
				// 跳过 pattern 中紧跟在 ** 之后的 '/'
				if pi < len(pattern) && pattern[pi] == '/' {
					pi++
				}
				// 跳过 target 当前段剩余部分
				for ti < len(target) && target[ti] != '/' {
					ti++
				}
				continue
			}
			// *: 单段 — 跳过当前段
			pi++ //  consume '*'
			if pi < len(pattern) && pattern[pi] == '/' {
				pi++ // consume following '/'
			}
			for ti < len(target) && target[ti] != '/' {
				ti++
			}
			matchIdx = ti
			continue
		}

		if pi < len(pattern) && pattern[pi] == '/' && ti < len(target) && target[ti] == '/' {
			pi++
			ti++
			continue
		}

		// 匹配失败：尝试回溯到最近的 **
		if starIdx != -1 {
			pi = starIdx + 2 // 回到 ** 之后
			if pi < len(pattern) && pattern[pi] == '/' {
				pi++
			}
			// 推进 target 越过一段
			matchIdx++
			ti = matchIdx
			for ti < len(target) && target[ti] != '/' {
				ti++
			}
			continue
		}

		return false
	}

	// target 已消费完，pattern 剩余部分只允许全是 ** 或分隔符
	for pi < len(pattern) {
		if pattern[pi] == '*' && pi+1 < len(pattern) && pattern[pi+1] == '*' {
			pi += 2
			if pi < len(pattern) && pattern[pi] == '/' {
				pi++
			}
			continue
		}
		if pattern[pi] == '*' {
			pi++
			if pi < len(pattern) && pattern[pi] == '/' {
				pi++
			}
			continue
		}
		return false
	}

	return true
}
```

- [ ] **Step 2: 删除 DPMatch() 函数和 matchDP 相关代码**

`match/match.go` 中移除 `DPMatch`。`match/segment.go` 中的 `segs` 类型和 `ParseSegs` 保留（供高级用户使用）。但 `Match()` 内部不再使用它们。

- [ ] **Step 3: 运行 match 包测试和 benchmark**

```bash
cd "D:\Code\Golang\Auther" && go test ./match/ -v
cd "D:\Code\Golang\Auther" && go test -bench=BenchmarkMatch -benchtime=200ms
```

Expected: 所有测试 PASS。`BenchmarkMatchExact` ~3ns，`BenchmarkMatchDoubleStar` ~20ns（零分配后大幅下降）。

- [ ] **Step 4: 如迭代算法有失败用例，调试修复**

重点检查：
- `/a/**/z` 匹配 `/a/z` 和 `/a/b/c/z`
- `/**` 匹配任意路径
- `*` 仅匹配单段（不跨越 `/`）
- 不存在通配符时字面量不匹配直接返回 false

---

### Task 5: 实现 Grants 索引

**Files:**
- Modify: `model/model.go` (RoleNode 新增字段)
- Modify: `auther.go` (buildTree 维护索引)
- Modify: `grant.go` (GrantResource/RevokeResource 维护索引)
- Modify: `enforce.go` (Enforce 使用索引)

- [ ] **Step 1: RoleNode 新增 GrantedMap 索引字段**

在 `model/model.go` 的 `RoleNode` 结构体中添加：

```go
type RoleNode struct {
	ID         string
	Parent     *RoleNode
	Children   map[string]*RoleNode
	Resources  map[string]bool
	GrantedMap map[string]bool // 索引：GrantedMap[resource] = true 表示 GrantsIn 中有该资源
	GrantsIn   []RoleGrant
	GrantsOut  []RoleGrant
	Users      map[string]*UserNode
}
```

- [ ] **Step 2: auther.go 中 NewAuthorizer 初始化 GrantedMap**

创建 root 角色时加上 `GrantedMap: make(map[string]bool)`：

```go
a.root = &model.RoleNode{
	ID:         "root",
	Children:   make(map[string]*model.RoleNode),
	Resources:  map[string]bool{"/**": true},
	GrantedMap: make(map[string]bool),
	Users:      make(map[string]*model.UserNode),
}
```

- [ ] **Step 3: buildTree Phase 1 初始化 GrantedMap**

```go
role := &model.RoleNode{
	ID:         rs.ID,
	Children:   make(map[string]*model.RoleNode),
	Resources:  make(map[string]bool),
	GrantedMap: make(map[string]bool),
	Users:      make(map[string]*model.UserNode),
}
```

- [ ] **Step 4: buildTree Phase 5 维护 GrantedMap**

添加授权记录后同步维护索引：

```go
grant := model.RoleGrant{FromRoleID: gs.FromRoleID, ToRoleID: gs.ToRoleID, Resource: gs.Resource}
fromRole.GrantsOut = append(fromRole.GrantsOut, grant)
toRole.GrantsIn = append(toRole.GrantsIn, grant)
toRole.GrantedMap[gs.Resource] = true // 维护索引
```

- [ ] **Step 5: GrantResource 维护索引**

```go
// GrantResource 成功添加 grant 后
toRole.GrantedMap[resource] = true
```

- [ ] **Step 6: RevokeResource 维护索引**

撤销授权时更新 GrantedMap。在移除 GrantsIn 条目时，检查是否还有同资源的其他授权决定是否从 GrantedMap 中删除：

```go
// 移除 grant 之后
toRole.GrantsIn = append(toRole.GrantsIn[:i], toRole.GrantsIn[i+1:]...)
// 检查是否还有同资源的其他授权
stillHas := false
for _, g := range toRole.GrantsIn {
	if g.Resource == resource {
		stillHas = true
		break
	}
}
if !stillHas {
	delete(toRole.GrantedMap, resource)
}
```

- [ ] **Step 7: role.go DeleteRole 维护 GrantedMap**

DeleteRole 中 filterGrantsByFrom/filterGrantsByTo 后也需重建受影响角色的 GrantedMap。在 filter 后添加重建逻辑。

- [ ] **Step 8: Enforce 使用 GrantedMap 快速路径**

```go
func (a *Authorizer) Enforce(userID, res string) (bool, error) {
	normalized, err := normalizeResource(res)
	if err != nil {
		return false, err
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	user := a.users[userID]
	if user == nil {
		return false, fmt.Errorf("%w: %s", ErrUserNotFound, userID)
	}
	role := user.Role
	if role == nil {
		return false, fmt.Errorf("auther: user %s has no role — corrupted state", userID)
	}

	// 快速路径：GrantedMap O(1) 查找
	if role.GrantedMap[normalized] {
		return true, nil
	}

	// 遍历角色自身资源
	for pattern := range role.Resources {
		if match.Match(pattern, normalized) {
			return true, nil
		}
	}

	// 线性扫描 GrantsIn（作为 GrantedMap 的兜底 — 处理通配符模式匹配）
	for _, g := range role.GrantsIn {
		if match.Match(g.Resource, normalized) {
			return true, nil
		}
	}
	return false, nil
}
```

注意：GrantedMap 索引只映射精确匹配的资源键，对于通配符模式（如 `/user/*`），仍需通过 GrantsIn 的 match 来检查。但 GrantedMap 中如果某 grant 是精确字符串（如 `/g/something`），则可以直接命中。

- [ ] **Step 9: 运行全量测试**

```bash
cd "D:\Code\Golang\Auther" && go test ./...
```

---

### Task 6: 添加 LRU 匹配缓存

**Files:**
- Modify: `model/model.go` (RoleNode 新增缓存字段)
- Modify: `enforce.go` (Enforce 使用缓存)
- Modify: `grant.go` (写操作清空缓存)

- [ ] **Step 1: RoleNode 新增 matchCache 字段**

```go
type RoleNode struct {
	// ... 其他字段
	matchCache map[string]bool // 匹配结果缓存（LRU 简化版）
}
```

注意小写开头 — 不导出，仅本包内部使用。

- [ ] **Step 2: 初始化 matchCache**

在 NewAuthorizer（root 创建）和 buildTree（Phase 1）中添加 `matchCache: make(map[string]bool)`。

- [ ] **Step 3: Enforce 使用缓存**

```go
// 在 Enforce 中，before 遍历 Resources 之前：
cacheKey := role.ID + "|" + normalized
if result, ok := role.GetMatchCache(cacheKey); ok {
	return result, nil
}

// ... 原有匹配逻辑 ...

// 匹配结束后存储结果
role.SetMatchCache(cacheKey, result)
```

简化版缓存（无 LRU 淘汰，容量用计数器限制）：

在 `model/model.go` 添加辅助方法：

```go
const maxCacheSize = 64

func (r *RoleNode) GetMatchCache(key string) (bool, bool) {
	v, ok := r.matchCache[key]
	return v, ok
}

func (r *RoleNode) SetMatchCache(key string, val bool) {
	if len(r.matchCache) >= maxCacheSize {
		// 超过容量时清空全部（简化版 LRU）
		r.matchCache = make(map[string]bool)
	}
	r.matchCache[key] = val
}
```

- [ ] **Step 4: 写操作清空缓存**

在 `GrantResource`、`RevokeResource`、`DeleteRole` 中，对涉及的 role 调用 `role.matchCache = make(map[string]bool)`（或设 nil）。
在 `buildTree` 中导出 `ResetCache` 方法。

- [ ] **Step 5: 运行测试 + benchmark**

```bash
cd "D:\Code\Golang\Auther" && go test ./... -v
cd "D:\Code\Golang\Auther" && go test -bench=BenchmarkEnforce -benchtime=500ms
```

Expected: 同上，BenmarkEnforce* 有 10-20% 额外改善（高频重复检查场景）。

---

### Task 7: 创建 examples/ 目录

**Files:**
- Create: `examples/basic/main.go`

- [ ] **Step 1: 创建 examples/basic/main.go**

```go
// 基本使用示例：创建角色、用户、授权，执行权限检查。
package main

import (
	"fmt"
	"auther"
	"auther/adapters/memory"
)

func main() {
	adapter := memoryadapter.NewMemoryAdapter()
	a, err := auther.NewAuthorizer(adapter)
	if err != nil {
		panic(err)
	}

	// 创建角色层级：root -> admin -> editor
	_ = a.CreateRole("root", "admin")
	_ = a.CreateRole("admin", "editor")

	// 向 admin 授予 /user/* 权限
	_ = a.GrantResource("admin", "admin", "/user/*")
	// 向 editor 授予 /data/* 权限
	_ = a.GrantResource("editor", "editor", "/data/*")
	// root 向 admin 授权 /g/**
	_ = a.GrantResource("root", "admin", "/g/**")

	// 创建用户
	_ = a.CreateUser("admin", "u_admin")
	_ = a.CreateUser("editor", "u_editor")

	// 权限检查
	checks := []struct{ user, resource string }{
		{"u_admin", "/user/create"},
		{"u_admin", "/g/anything"},
		{"u_editor", "/data/read"},
		{"u_editor", "/user/create"},
	}

	for _, c := range checks {
		ok, _ := a.Enforce(c.user, c.resource)
		fmt.Printf("Enforce(%s, %s) = %v\n", c.user, c.resource, ok)
	}
}
```

- [ ] **Step 2: 编译并运行示例**

```bash
cd "D:\Code\Golang\Auther" && go run ./examples/basic/
```

Expected: 输出 4 行权限检查结果。

---

### Task 8: adapter.go 适配 update + 最终验证

**Files:**
- Modify: `adapter.go` (确保接口使用 model 包类型，类型别名保证兼容)
- Modify: `adapters/memory/memory_adapter.go` (import 更新)
- Modify: `adapters/file/file_adapter.go` (import 更新)

- [ ] **Step 1: 验证 adapter.go 接口兼容性**

```go
package auther

import "auther/model"

// Adapter 定义了 Auther 策略持久化的接口。
type Adapter interface {
	Load() (*model.PolicySnapshot, error)
	Save(snapshot *model.PolicySnapshot) error
}
```

用 `model.PolicySnapshot` 替代直接引用（类型别名 `PolicySnapshot = model.PolicySnapshot` 仍然有效）。

- [ ] **Step 2: 运行全量测试和 vet**

```bash
cd "D:\Code\Golang\Auther" && go vet ./... && go test ./...
```

Expected: vet 零警告，全部测试 PASS。

- [ ] **Step 3: 运行全量 benchmark，确认性能改善**

```bash
cd "D:\Code\Golang\Auther" && go test -bench=. -benchtime=500ms ./...
```

记录 benchmark 结果与优化前对比。

- [ ] **Step 4: 更新 README.md 中的示例代码**

确保 README 中的 import 路径和用法与新结构一致。

---

### Task 9: 最终修复和清理

- [ ] **Step 1: 检查所有文件中 import 路径**

```bash
cd "D:\Code\Golang\Auther" && grep -rn "auther\"" --include="*.go" .
```

确保：
- 根包内文件没有 import "auther"（Go 不允许同包 import）
- adapters 正确 import "auther" 和 "auther/model"
- match 包和 model 包没有 import "auther"

- [ ] **Step 2: 运行 go mod tidy**

```bash
cd "D:\Code\Golang\Auther" && go mod tidy
```

- [ ] **Step 3: 最终全量测试**

```bash
cd "D:\Code\Golang\Auther" && go vet ./... && go test ./... -count=1
```

Expected: 全部 PASS。
