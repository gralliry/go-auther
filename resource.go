package auther

import (
	"fmt"

	"github.com/gralliry/go-auther/internal/resource"
)

// normalizeRes 调用 resource.New 并封装错误为 ErrInvalidResource。
func normalizeRes(raw string) (resource.Resource, error) {
	res, err := resource.New(raw)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidResource, err)
	}
	return res, nil
}
