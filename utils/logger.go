package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	logFile    *os.File
	logger     *log.Logger
	mu         sync.Mutex
	clients    = make(map[*websocket.Conn]bool)
	clientsMux sync.Mutex
)

// InitLogger 初始化日志系统
func InitLogger() error {
	mu.Lock()
	defer mu.Unlock()

	if logger != nil {
		return nil
	}

	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	logPath := filepath.Join(logDir, "app.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	logFile = file
	// 同时输出到文件和控制台
	mw := io.MultiWriter(os.Stdout, file)
	logger = log.New(mw, "", 0)

	// 启动日志轮转检查
	go checkLogRotation()

	return nil
}

// LogError 记录错误信息
func LogError(message string, err error) {
	if logger == nil {
		InitLogger()
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	errMsg := fmt.Sprintf("[%s] [ERROR] %s", timestamp, message)
	if err != nil {
		errMsg += fmt.Sprintf(": %v", err)
	}
	logger.Println(errMsg)
	BroadcastLog(errMsg)
}

// LogInfo 记录信息日志
func LogInfo(message string) {
	if logger == nil {
		InitLogger()
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMessage := fmt.Sprintf("[%s] [INFO] %s\n", timestamp, message)
	logger.Print(logMessage)
	BroadcastLog(logMessage)
}

// checkLogRotation 检查并执行日志轮转
func checkLogRotation() {
	for {
		time.Sleep(time.Hour) // 每小时检查一次
		if needRotation() {
			rotateLog()
		}
	}
}

// needRotation 检查是否需要轮转
func needRotation() bool {
	if logFile == nil {
		return false
	}

	info, err := logFile.Stat()
	if err != nil {
		return false
	}

	// 如果日志文件大于10MB，进行轮转
	return info.Size() > 10*1024*1024
}

// rotateLog 执行日志轮转
func rotateLog() {
	mu.Lock()
	defer mu.Unlock()

	if logFile == nil {
		return
	}

	logFile.Close()

	// 重命名当前日志文件
	oldPath := filepath.Join("logs", "app.log")
	newPath := filepath.Join("logs", fmt.Sprintf("app.%s.log",
		time.Now().Format("20060102150405")))

	os.Rename(oldPath, newPath)

	// 创建新的日志文件
	InitLogger()
}

// BroadcastLog 向所有连接的WebSocket客户端广播日志
func BroadcastLog(message string) {
	clientsMux.Lock()
	defer clientsMux.Unlock()

	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			client.Close()
			delete(clients, client)
		}
	}
}

// AddClient 添加新的WebSocket客户端
func AddClient(conn *websocket.Conn) {
	clientsMux.Lock()
	clients[conn] = true
	clientsMux.Unlock()
}

// RemoveClient 移除WebSocket客户端
func RemoveClient(conn *websocket.Conn) {
	clientsMux.Lock()
	delete(clients, conn)
	clientsMux.Unlock()
}
