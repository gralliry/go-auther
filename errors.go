package auther

import "errors"

// 预定义的哨兵错误，调用方可通过 errors.Is 进行判断。
var (
	// ErrUserNotFound 用户不存在。
	ErrUserNotFound = errors.New("auther: user not found")

	// ErrRoleNotFound 角色不存在。
	ErrRoleNotFound = errors.New("auther: role not found")

	// ErrGrantNotFound 授权记录不存在。
	ErrGrantNotFound = errors.New("auther: grant not found")

	// ErrNotAncestor 授权方不是接收方的祖先角色，授权被拒绝。
	ErrNotAncestor = errors.New("auther: grant target must be a descendant of the grantor role")

	// ErrCircularRoleHierarchy 角色层级中存在循环引用。
	ErrCircularRoleHierarchy = errors.New("auther: circular role hierarchy detected")

	// ErrInvalidResource 资源路径格式无效。
	ErrInvalidResource = errors.New("auther: invalid resource pattern")

	// ErrDuplicateUser 尝试创建已存在的用户。
	ErrDuplicateUser = errors.New("auther: user already exists")

	// ErrDuplicateRole 尝试创建已存在的角色。
	ErrDuplicateRole = errors.New("auther: role already exists")

	// ErrDuplicateGrant 尝试创建已存在的授权记录。
	ErrDuplicateGrant = errors.New("auther: grant already exists")

	// ErrRootRoleDelete 禁止删除根角色。
	ErrRootRoleDelete = errors.New("auther: cannot delete root role")
)
