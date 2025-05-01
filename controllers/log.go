package controllers

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"backend/utils"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// LogResponse 日志响应结构
type LogResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境应该更严格
	},
	HandshakeTimeout: 10 * time.Second,
}

// initLogFile 初始化日志文件和目录
func initLogFile(logPath string) error {
	dir := filepath.Dir(logPath)

	// 创建日志目录
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 检查日志文件是否存在，不存在则创建
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		file, err := os.Create(logPath)
		if err != nil {
			return err
		}
		file.Close()
	}
	return nil
}

// GetLogs godoc
// @Summary      获取系统日志
// @Description  获取系统日志文件的最新内容
// @Tags         系统管理
// @Accept       json
// @Produce      json
// @Param        lines  query    int     false  "返回的日志行数(默认100)"  minimum(1) maximum(1000)
// @Success      200    {object} LogResponse
// @Failure      500    {object} LogResponse
// @Security     Bearer
// @Router       /admin/logs [get]
func GetLogs(c *gin.Context) {
	lines := 100 // 默认返回100行
	if lineParam := c.Query("lines"); lineParam != "" {
		if parsedLines, err := strconv.Atoi(lineParam); err == nil && parsedLines > 0 && parsedLines <= 1000 {
			lines = parsedLines
		}
	}

	logPath := filepath.Join("logs", "app.log")

	// 确保日志文件存在
	if err := initLogFile(logPath); err != nil {
		c.JSON(http.StatusInternalServerError, LogResponse{
			Code:    http.StatusInternalServerError,
			Message: "初始化日志文件失败",
			Error:   err.Error(),
		})
		return
	}

	file, err := os.Open(logPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, LogResponse{
			Code:    http.StatusInternalServerError,
			Message: "无法访问日志文件",
			Error:   err.Error(),
		})
		return
	}
	defer file.Close()

	// 读取最后N行日志
	var logLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		logLines = append(logLines, scanner.Text())
		if len(logLines) > lines {
			logLines = logLines[1:]
		}
	}

	if err := scanner.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, LogResponse{
			Code:    http.StatusInternalServerError,
			Message: "读取日志文件失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, LogResponse{
		Code:    http.StatusOK,
		Message: "获取日志成功",
		Data:    logLines,
	})
}

// WatchLogs godoc
// @Summary      实时监控系统日志
// @Description  通过WebSocket实时接收系统日志
// @Tags         系统管理
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Router       /admin/logs/watch [get]
func WatchLogs(c *gin.Context) {
	// 记录连接尝试
	utils.LogInfo(fmt.Sprintf("WebSocket连接尝试 - IP: %s", c.ClientIP()))

	// 升级HTTP连接为WebSocket连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		utils.LogError("升级WebSocket连接失败", err)
		return
	}

	// 1. 认证：从URL参数获取token
	token := c.Query("token")
	if token == "" {
		conn.WriteJSON(map[string]interface{}{
			"type":    "auth_error",
			"message": "缺少token",
		})
		conn.Close()
		return
	}

	// 2. 校验token
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !parsedToken.Valid {
		conn.WriteJSON(map[string]interface{}{
			"type":    "auth_error",
			"message": "无效的token",
		})
		conn.Close()
		return
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		conn.WriteJSON(map[string]interface{}{
			"type":    "auth_error",
			"message": "无效的token claims",
		})
		conn.Close()
		return
	}

	// 3. 检查角色
	role, _ := claims["role"].(string)
	if role != "admin" {
		conn.WriteJSON(map[string]interface{}{
			"type":    "auth_error",
			"message": "权限不足",
		})
		conn.Close()
		return
	}

	// 4. 认证通过
	conn.WriteJSON(map[string]interface{}{
		"type":    "auth_success",
		"message": "认证成功",
	})

	// 添加客户端到广播列表
	utils.AddClient(conn)

	// 记录连接成功
	utils.LogInfo(fmt.Sprintf("WebSocket连接成功 - IP: %s", c.ClientIP()))

	// 设置连接参数
	conn.SetReadLimit(512) // 设置消息大小限制
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 当连接关闭时移除客户端
	defer func() {
		utils.RemoveClient(conn)
		conn.Close()
		utils.LogInfo(fmt.Sprintf("WebSocket连接关闭 - IP: %s", c.ClientIP()))
	}()

	// 启动心跳检测
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	// 保持连接活跃并处理消息
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				utils.LogError("WebSocket读取错误", err)
			}
			break
		}
	}
}
