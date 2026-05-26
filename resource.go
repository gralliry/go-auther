package auther

import (
	"fmt"

	"github.com/gralliry/go-auther/internal/match"
)

// normalizeRes 调用 match.Clean 并封装错误为 ErrInvalidResource。
func normalizeRes(raw string) (string, error) {
	res, err := match.Clean(raw)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidResource, err)
	}
	return res, nil
}
