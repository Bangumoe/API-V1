package controllers

import (
	"backend/config"
	"backend/services/mail"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// MailSettingsRequest 邮件设置请求结构
// @Description 邮件服务器配置请求结构
type MailSettingsRequest struct {
	Host        string `json:"host" binding:"required" example:"smtp.gmail.com" description:"SMTP服务器地址"`
	Port        int    `json:"port" binding:"required" example:"587" description:"SMTP服务器端口"`
	Username    string `json:"username" binding:"required" example:"your-email@gmail.com" description:"SMTP用户名"`
	Password    string `json:"password" example:"your-password" description:"SMTP密码（如不修改可留空）"`
	FromAddress string `json:"from_address" binding:"required,email" example:"noreply@example.com" description:"发件人邮箱地址"`
	FromName    string `json:"from_name" binding:"required" example:"动画网站" description:"发件人显示名称"`
	UseTLS      bool   `json:"use_tls" example:"true" description:"是否使用TLS加密"`
}

// TestMailRequest 测试邮件请求结构
// @Description 发送测试邮件的请求结构
type TestMailRequest struct {
	To string `json:"to" binding:"required,email" example:"test@example.com" description:"测试邮件接收地址"`
}

// SendCustomMailRequest 自定义邮件请求结构
// @Description 发送自定义邮件的请求结构
type SendCustomMailRequest struct {
	To      []string `json:"to" binding:"required,dive,email" example:"user1@example.com,user2@example.com" description:"收件人邮箱地址列表"`
	Subject string   `json:"subject" binding:"required" example:"重要通知" description:"邮件主题"`
	Content string   `json:"content" binding:"required" example:"<h1>网站更新通知</h1><p>内容...</p>" description:"邮件内容"`
	IsHTML  bool     `json:"is_html" example:"true" description:"是否为HTML格式内容"`
}

// MailSettingsResponse 邮件设置响应结构
// @Description 邮件服务器配置响应结构
type MailSettingsResponse struct {
	Host        string `json:"host" example:"smtp.gmail.com" description:"SMTP服务器地址"`
	Port        int    `json:"port" example:"587" description:"SMTP服务器端口"`
	Username    string `json:"username" example:"your-email@gmail.com" description:"SMTP用户名"`
	FromAddress string `json:"from_address" example:"noreply@example.com" description:"发件人邮箱地址"`
	FromName    string `json:"from_name" example:"动画网站" description:"发件人显示名称"`
	UseTLS      bool   `json:"use_tls" example:"true" description:"是否使用TLS加密"`
}

type MailSettingsController struct {
	DB *gorm.DB
}

func NewMailSettingsController(db *gorm.DB) *MailSettingsController {
	return &MailSettingsController{DB: db}
}

// GetMailSettings godoc
// @Summary      获取邮件服务设置
// @Description  获取系统的邮件服务配置信息
// @Tags         邮件管理
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Success      200  {object}  MailSettingsResponse
// @Failure      401  {object}  Response
// @Router       /admin/mail/settings [get]
func (mc *MailSettingsController) GetMailSettings(c *gin.Context) {
	cfg := config.GetConfig()
	// 不返回密码
	mailSettings := struct {
		Host        string `json:"host"`
		Port        int    `json:"port"`
		Username    string `json:"username"`
		FromAddress string `json:"from_address"`
		FromName    string `json:"from_name"`
		UseTLS      bool   `json:"use_tls"`
	}{
		Host:        cfg.Mail.Host,
		Port:        cfg.Mail.Port,
		Username:    cfg.Mail.Username,
		FromAddress: cfg.Mail.FromAddress,
		FromName:    cfg.Mail.FromName,
		UseTLS:      cfg.Mail.UseTLS,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    mailSettings,
	})
}

// UpdateMailSettings godoc
// @Summary      更新邮件服务设置
// @Description  更新系统的邮件服务配置信息
// @Tags         邮件管理
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        body  body      MailSettingsRequest  true  "邮件设置参数"
// @Success      200  {object}  Response{message=string}
// @Failure      400  {object}  Response{message=string,error=string}
// @Failure      401  {object}  Response
// @Failure      500  {object}  Response{message=string,error=string}
// @Router       /admin/mail/settings [put]
func (mc *MailSettingsController) UpdateMailSettings(c *gin.Context) {
	var req MailSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数无效",
			"error":   err.Error(),
		})
		return
	}

	cfg := config.GetConfig()
	cfg.Mail.Host = req.Host
	cfg.Mail.Port = req.Port
	cfg.Mail.Username = req.Username
	if req.Password != "" {
		cfg.Mail.Password = req.Password
	}
	cfg.Mail.FromAddress = req.FromAddress
	cfg.Mail.FromName = req.FromName
	cfg.Mail.UseTLS = req.UseTLS

	if err := config.SaveConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "保存配置失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "邮件设置更新成功",
	})
}

// TestMailSettings godoc
// @Summary      发送测试邮件
// @Description  使用当前邮件服务设置发送测试邮件
// @Tags         邮件管理
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        body  body      TestMailRequest  true  "测试邮件参数"
// @Success      200  {object}  Response{message=string}
// @Failure      400  {object}  Response{message=string,error=string}
// @Failure      401  {object}  Response
// @Failure      500  {object}  Response{message=string,error=string}
// @Router       /admin/mail/test [post]
func (mc *MailSettingsController) TestMailSettings(c *gin.Context) {
	var req TestMailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数无效",
			"error":   err.Error(),
		})
		return
	}

	mailService := mail.NewMailService()
	err := mailService.SendMail(req.To, "测试邮件", `
		<h2>邮件发送测试</h2>
		<p>这是一封测试邮件，用于验证邮件服务配置是否正确。</p>
		<p>如果您收到这封邮件，说明邮件服务配置成功！</p>
	`)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "发送测试邮件失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "测试邮件发送成功",
	})
}

// SendCustomMail godoc
// @Summary      发送自定义邮件
// @Description  发送自定义邮件到指定邮箱
// @Tags         邮件管理
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        body  body      SendCustomMailRequest  true  "自定义邮件内容"
// @Success      200  {object}  Response{message=string}
// @Failure      400  {object}  Response{message=string,error=string}
// @Failure      401  {object}  Response
// @Failure      500  {object}  Response{message=string,error=string}
// @Router       /admin/mail/send [post]
func (mc *MailSettingsController) SendCustomMail(c *gin.Context) {
	var req SendCustomMailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数无效",
			"error":   err.Error(),
		})
		return
	}

	mailService := mail.NewMailService()
	err := mailService.SendCustomMail(req.To, req.Subject, req.Content, req.IsHTML)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "发送邮件失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "邮件发送成功",
	})
}
