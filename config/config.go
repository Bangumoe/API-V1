package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type MailConfig struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	FromAddress string `json:"from_address"`
	FromName    string `json:"from_name"`
	UseTLS      bool   `json:"use_tls"`
}

type Config struct {
	IsBetaMode bool       `json:"is_beta_mode"`
	Mail       MailConfig `json:"mail"`
}

var (
	config *Config
	once   sync.Once
)

func GetConfig() *Config {
	once.Do(func() {
		config = &Config{
			IsBetaMode: false, // 默认关闭内测模式
			Mail: MailConfig{
				Host:        "",
				Port:        587,
				Username:    "",
				Password:    "",
				FromAddress: "",
				FromName:    "动画网站",
				UseTLS:      true,
			},
		}
		loadConfig()
	})
	return config
}

func loadConfig() {
	// 确保config目录存在
	if err := os.MkdirAll("config", 0755); err != nil {
		return
	}

	file, err := os.Open("config/config.json")
	if err != nil {
		return
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		return
	}
}

func SaveConfig() error {
	// 确保config目录存在
	if err := os.MkdirAll("config", 0755); err != nil {
		return err
	}

	// 创建临时文件
	tmpFile := filepath.Join("config", "config.json.tmp")
	file, err := os.Create(tmpFile)
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile) // 清理临时文件

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(config); err != nil {
		file.Close()
		return err
	}
	file.Close()

	// 原子性地重命名临时文件
	return os.Rename(tmpFile, "config/config.json")
}

func SetBetaMode(enabled bool) error {
	cfg := GetConfig() // 确保config已经被初始化
	cfg.IsBetaMode = enabled
	return SaveConfig()
}
