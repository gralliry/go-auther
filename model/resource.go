package model

import (
	"fmt"
	"path"

	"auther/match"
)

// Resource 是一个经过校验和规范化的资源路径模式。
// 支持 *（匹配单个路径段）和 **（匹配零个或多个路径段）通配符。
type Resource string

// NewResource 校验并规范化一个原始资源路径。
func NewResource(raw string) (Resource, error) {
	if raw == "" {
		return "", fmt.Errorf("resource must not be empty")
	}
	if raw[0] != '/' {
		return "", fmt.Errorf("resource must start with '/'")
	}
	return Resource(path.Clean(raw)), nil
}

// Match 判断 target 是否匹配此资源模式。
func (r Resource) Match(target string) bool {
	return match.Match(string(r), target)
}

// HasWildcard 判断此模式是否包含 '*' 通配符。
func (r Resource) HasWildcard() bool {
	return match.HasWildcard(string(r))
}

// String 返回规范化后的路径字符串。
func (r Resource) String() string { return string(r) }
