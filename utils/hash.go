package utils

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

// CalculateFileMD5 计算文件MD5哈希值
func CalculateFileMD5(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}
	return hex.EncodeToString(hash.Sum(nil))
}
