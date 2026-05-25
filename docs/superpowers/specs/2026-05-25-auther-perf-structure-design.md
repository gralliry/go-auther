# Auther 性能优化与项目结构规范设计

日期：2026-05-25

## 背景

当前 auther 权限库所有代码集中在根目录单一 package，文件间职责模糊，内部实现（parseSegs、matchDP）对外可见。Enforce 热路径存在可避免的内存分配，Grants 使用线性扫描导致高频读取场景下性能不佳。

## 目标

1. 优化 Enforce 热路径，将延迟从 ~400ns 降至 ~100ns
2. 规范项目结构，按功能拆分子包，消除循环依赖
3. 外部 API 保持 `import "auther"` 不变

## 使用场景

高频读取、低频写入（类数据库权限系统）。每秒数千次 Enforce 调用，角色/用户变更很少。优化重心在读取路径。

---

## 一、项目结构

### 新布局

```
auther/                       ← 主包（公开 API + 内部实现）
├── model/                    ← 数据模型（零依赖）
│   └── model.go              RoleNode, UserNode, RoleGrant, PolicySnapshot 等
│
├── match/                    ← 匹配引擎（零依赖）
│   ├── match.go              Match(), HasWildcard()
│   └── segment.go            ParseSegs(), segs 类型, DPMatch()
│
├── adapters/
│   ├── memory/               内存适配器
│   └── file/                 JSON 文件适配器
│
├── examples/
│   └── basic/
│       └── main.go           基本使用示例
│
├── auther.go                 Authorizer, NewAuthorizer, buildTree, 树遍历
├── enforce.go                Enforce, GetUserPermissions
├── grant.go                  GrantResource, RevokeResource, 授权查询
├── role.go                   CreateRole, DeleteRole, 角色查询
├── user.go                   CreateUser, DeleteUser, 用户查询
├── resource.go               normalizeResource（调用 match 包）
├── errors.go                 哨兵错误定义
├── adapter.go                Adapter 接口
├── go.mod
└── README.md
```

### 依赖方向（单向无循环）

```
model  ←  match  ←  auther  ←  adapters/{memory,file}
(零依赖)   (零依赖)   (公开API)    (实现 Adapter 接口)
```

### 拆分原则

| 包 | 内容 | 对外可见性 |
|---|------|-----------|
| `auther` | Authorizer、公开方法、错误、Adapter 接口 | 全部公开 |
| `auther/model` | RoleNode、UserNode、RoleGrant、Snapshot DTO | 全部公开（供适配器使用） |
| `auther/match` | Match()、ParseSegs()、DPMatch() | 全部公开（供高级用户自定义匹配） |
| `auther/adapters/*` | Adapter 实现 | 全部公开 |

### 类型别名

主包中使用 type alias 简化常用类型引用：

```go
// auther/auther.go
type RoleGrant = model.RoleGrant
type RoleInfo   = model.RoleInfo
type UserInfo   = model.UserInfo
```

外部用户 import "auther" 即可使用所有公开类型，无需额外 import "auther/model"。

---

## 二、性能优化

### 2.1 match 零分配（优先级最高）

**当前问题**：每次 match 调用分配 segs 结构中的两个 slice（p, on）+ matchDP 中的 DP 表。

**方案**：
- 用迭代算法替代 DP 表分配。对于 `*` 和 `**` 匹配，使用双指针回溯算法（参考 Unix glob 实现），无需任何堆分配
- parseSegs 改为返回栈分配的 segs 视图，或提供带缓存的版本

**预期**：Match 从 300ns → 50ns，零分配

### 2.2 Grants 索引化

**当前问题**：Enforce 中遍历 `role.GrantsIn` 切片，每个授权都调用 match()，复杂度 O(n×m)。

**方案**：在 RoleNode 中新增 `GrantedResources map[string]bool`，写操作时维护此索引。Enforce 时直接查 map，O(1)。

**预期**：HitGrant 从 719ns → 400ns（EnforceMissAll 类似降幅）

### 2.3 资源编译缓存

**当前问题**：Enforce 每次对同一角色+资源组合重新 match。

**方案**：
- 为 RoleNode 添加 LRU 缓存（容量 64），缓存最近 match 结果
- 写操作（Grant/Revoke）时清空缓存

**预期**：重复检查 → 50ns（缓存命中）

### 2.4 读写锁优化

**当前问题**：单个 RWMutex 保护所有操作。

**方案**：
- 将 roles/users map 的并发控制改用 `sync.Map`（读优化）或分段锁
- 或保持 RWMutex 但减少临界区范围

**风险评估**：sync.Map 适合读多写少场景，但 API 与普通 map 不同，需要评估代码改动量。此优化收益中等，可作为可选项。

**预期**：读延迟降低 20-30%

### 2.5 写操作异步化（可选）

**当前问题**：每次写操作同步调用 adapter.Save()，阻塞调用方。

**方案**：
- 引入写缓冲区，写操作立即返回，后台 goroutine 批量刷盘
- 或至少让 adapter.Save() 在内部异步化

**风险评估**：改变持久化语义（write-through → write-back），可能丢失未刷盘的数据。需要与用户确认。

---

## 三、实施步骤

| 阶段 | 内容 | 预计影响文件 |
|------|------|-------------|
| 1 | 创建 `model/` 包，迁移数据类型 | types.go → model/model.go |
| 2 | 创建 `match/` 包，迁移匹配逻辑 | resource.go → match/ |
| 3 | 更新主包引用（model.RoleNode 等） | auther.go, enforce.go, grant.go, role.go, user.go |
| 4 | 添加主包类型别名 | auther.go |
| 5 | 实现迭代匹配算法（零分配） | match/match.go |
| 6 | 实现 Grants 索引 | types.go (RoleNode), enforce.go, grant.go, auther.go |
| 7 | 添加 LRU 缓存 | auther.go (RoleNode), enforce.go |
| 8 | 创建 examples/ 目录 | examples/basic/main.go |
| 9 | 更新 adapters/ 引用 | adapters/memory/, adapters/file/ |
| 10 | 运行全量测试 + benchmark | 所有包 |

---

## 四、对外 API 兼容性

拆分后 import 路径保持不变：

```go
import "auther"  // 不变，类型别名保证兼容
```

适配器实现需要额外 import：

```go
import (
    "auther"
    "auther/model"  // 访问 PolicySnapshot 等 DTO
)
```

公开方法签名全部不变，类型别名保证 `auther.RoleGrant` 等引用仍然有效。
