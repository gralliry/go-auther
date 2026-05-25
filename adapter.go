package auther

import "auther/model"

// Adapter 定义了 Auther 策略持久化的接口。
//
// 实现必须是并发安全的。Load 方法在 Authorizer 构造时调用一次；
// Save 在每次数据变更后调用（写透模式）。
type Adapter interface {
	// Load 从存储中加载完整的策略快照。
	// 如果尚无数据存在，应返回 nil 快照且不报错。
	Load() (*model.PolicySnapshot, error)

	// Save 将完整的策略快照持久化到存储中。
	// 实现应使用原子写入（例如临时文件 + 重命名）以防止数据损坏。
	Save(snapshot *model.PolicySnapshot) error
}
