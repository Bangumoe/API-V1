package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const codeChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// GenerateRandomString 生成指定长度的随机字符串
// 在生产环境中，此函数现在返回一个错误，而不是在发生问题时 panic。
func GenerateRandomString(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be a positive integer")
	}
	b := make([]byte, length)
	max := big.NewInt(int64(len(codeChars)))
	for i := range b {
		val, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("failed to generate random number for string: %w", err)
		}
		b[i] = codeChars[val.Int64()]
	}
	return string(b), nil
}
