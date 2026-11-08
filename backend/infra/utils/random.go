package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// RandomCode 生成 prefix 加随机串的短码，用于自定义字典项。
func RandomCode(prefix string) string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return prefix + "00000000"
	}
	return prefix + hex.EncodeToString(buf)
}
