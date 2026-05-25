package auther

import "fmt"

// normalizeResource 校验并规范化资源路径。
//   - 不能为空
//   - 必须以 '/' 开头
//   - 合并连续的斜杠（// → /）
//   - 去除尾部斜杠（根路径 "/" 除外）
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
