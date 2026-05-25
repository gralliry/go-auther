package auther

import (
	"fmt"
	"path"
)

// normalizeResource 校验并规范化资源路径。
func normalizeResource(resource string) (string, error) {
	if resource == "" {
		return "", fmt.Errorf("%w: resource must not be empty", ErrInvalidResource)
	}
	if resource[0] != '/' {
		return "", fmt.Errorf("%w: resource must start with '/'", ErrInvalidResource)
	}
	return path.Clean(resource), nil
}
