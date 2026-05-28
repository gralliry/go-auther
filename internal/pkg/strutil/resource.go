package strutil

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func NormalizeKey(separator string, keys ...string) string {
	if len(keys) == 0 {
		return ""
	}
	str := strings.Join(keys, separator)
	if len(str) <= 64 {
		return "o" + str
	}
	sum := sha256.Sum256([]byte(str))
	return "h" + hex.EncodeToString(sum[:])
}
