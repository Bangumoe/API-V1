package controllers

import (
	"backend/models"
	"backend/services/mail"
	"backend/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// InvitationCodeController 处理邀请码相关的请求
type InvitationCodeController struct {
	db *gorm.DB
}

// NewInvitationCodeController 创建新的 InvitationCodeController 实例
func NewInvitationCodeController(db *gorm.DB) *InvitationCodeController {
	return &InvitationCodeController{db: db}
}

// GenerateInvitationCodesRequest 生成邀请码请求结构体
type GenerateInvitationCodesRequest struct {
	Count     int  `json:"count" binding:"required,min=1,max=100"`
	ExpiresIn *int `json:"expires_in,omitempty"` // 过期时间，单位：天
}

// GenerateInvitationCodes 生成邀请码
// @Summary 生成邀请码 (管理员)
// @Description 管理员生成指定数量的邀请码
// @Tags 邀请码管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body GenerateInvitationCodesRequest true "请求参数"
// @Success 200 {object} map[string]interface{} "返回生成的邀请码列表"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 403 {object} map[string]string "权限不足"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /admin/invitation-codes/generate [post]
func (icc *InvitationCodeController) GenerateInvitationCodes(c *gin.Context) {
	claims, _ := c.Get("claims")
	userClaims := claims.(jwt.MapClaims)
	adminID := uint(userClaims["user_id"].(float64))

	var req GenerateInvitationCodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数: " + err.Error()})
		return
	}

	var codes []models.InvitationCode
	var expiresAt *time.Time
	if req.ExpiresIn != nil {
		val := time.Now().AddDate(0, 0, *req.ExpiresIn)
		expiresAt = &val
	}

	for i := 0; i < req.Count; i++ {
		codeStr, err := utils.GenerateRandomString(16) // 生成16位随机码
		if err != nil {
			// 记录日志并返回服务器错误，因为这通常是环境或密码库问题
			utils.LogError("Failed to generate invitation code string", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "生成邀请码时发生内部错误"})
			return
		}
		newCode := models.InvitationCode{
			Code:        codeStr,
			IsUsed:      false,
			ExpiresAt:   expiresAt,
			GeneratedBy: &adminID,
		}
		codes = append(codes, newCode)
	}

	if err := icc.db.Create(&codes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成邀请码失败: " + err.Error()})
		return
	}

	var resultCodes []string
	for _, code := range codes {
		resultCodes = append(resultCodes, code.Code)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("成功生成 %d 个邀请码", req.Count),
		"codes":   resultCodes,
	})
}

// ListInvitationCodes 列出所有邀请码
// @Summary 列出所有邀请码 (管理员)
// @Description 管理员查看所有邀请码及其状态
// @Tags 邀请码管理
// @Produce json
// @Security Bearer
// @Success 200 {object} map[string]interface{} "返回邀请码列表"
// @Failure 403 {object} map[string]string "权限不足"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /admin/invitation-codes [get]
func (icc *InvitationCodeController) ListInvitationCodes(c *gin.Context) {
	var codes []models.InvitationCode
	// 定义需要选择的用户字段
	userSelectFields := []string{"id", "username", "email", "role", "avatar", "is_allowed", "created_at", "updated_at"}

	if err := icc.db.Preload("UsedByUser", func(db *gorm.DB) *gorm.DB {
		return db.Select(userSelectFields)
	}).Preload("Generator", func(db *gorm.DB) *gorm.DB {
		return db.Select(userSelectFields)
	}).Order("created_at desc").Find(&codes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取邀请码列表失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": codes})
}

// DeleteInvitationCode 删除邀请码
// @Summary 删除邀请码 (管理员)
// @Description 管理员删除一个未使用的邀请码
// @Tags 邀请码管理
// @Produce json
// @Security Bearer
// @Param code path string true "邀请码"
// @Success 200 {object} map[string]string "返回操作结果"
// @Failure 400 {object} map[string]string "邀请码不存在或已被使用"
// @Failure 403 {object} map[string]string "权限不足"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /admin/invitation-codes/{code} [delete]
func (icc *InvitationCodeController) DeleteInvitationCode(c *gin.Context) {
	codeParam := c.Param("code")

	var invCode models.InvitationCode
	if err := icc.db.Where("code = ?", codeParam).First(&invCode).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "邀请码不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询邀请码失败: " + err.Error()})
		return
	}

	if invCode.IsUsed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邀请码已被使用，不能删除"})
		return
	}

	if err := icc.db.Delete(&invCode).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除邀请码失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "邀请码删除成功"})
}

// SendInvitationCodeRequest 发送邀请码请求结构体
type SendInvitationCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required"`
}

// SendInvitationCode godoc
// @Summary 发送邀请码
// @Description 将指定的邀请码发送到指定邮箱
// @Tags 邀请码管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body SendInvitationCodeRequest true "发送邀请码请求"
// @Success 200 {object} map[string]string "发送成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "邀请码不存在"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /admin/invitation-codes/send [post]
func (icc *InvitationCodeController) SendInvitationCode(c *gin.Context) {
	var req SendInvitationCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("无效的请求参数: %v", err),
		})
		return
	}

	// 验证邀请码是否存在且有效
	var invCode models.InvitationCode
	if err := icc.db.Where("code = ?", req.Code).First(&invCode).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "邀请码不存在",
			})
			return
		}
		utils.LogError("查询邀请码失败", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("查询邀请码失败: %v", err),
		})
		return
	}

	// 检查邀请码是否已使用
	if invCode.IsUsed {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "邀请码已被使用",
		})
		return
	}

	// 检查邀请码是否已过期
	if invCode.ExpiresAt != nil && invCode.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "邀请码已过期",
		})
		return
	}

	// 发送邮件
	mailService := mail.NewMailService()
	if err := mailService.SendInvitationCode(req.Email, req.Code, invCode.ExpiresAt); err != nil {
		utils.LogError("发送邀请码邮件失败", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("发送邀请码邮件失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "邀请码已成功发送",
	})
}
