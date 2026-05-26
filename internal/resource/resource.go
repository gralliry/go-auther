package resource

import (
	"fmt"
	"path"
)

// Resource 表示一个已验证并规范化的资源路径模式。
type Resource string

// New 校验并规范化一个资源路径，返回 Resource。
func New(raw string) (Resource, error) {
	if raw == "" {
		return "", fmt.Errorf("resource must not be empty")
	}
	if raw[0] != '/' {
		return "", fmt.Errorf("resource must start with '/'")
	}
	return Resource(path.Clean(raw)), nil
}

// String 返回资源的字符串形式。
func (r Resource) String() string { return string(r) }

// Match 判断 target 是否匹配当前资源模式的 glob 规则。
func (r Resource) Match(target string) bool {
	pattern := r.String()
	if pattern == target {
		return true
	}
	if !r.hasWildcard() {
		return false
	}
	return r.matchGlob(pattern, target)
}
