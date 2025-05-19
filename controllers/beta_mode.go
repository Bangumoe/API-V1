package controllers

import (
	"backend/config"
	"backend/models"

	"net/http"

	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// BetaModeController 处理内测模式相关的请求
type BetaModeController struct {
	db *gorm.DB
}

// NewBetaModeController 创建新的BetaModeController实例
func NewBetaModeController(db *gorm.DB) *BetaModeController {
	return &BetaModeController{db: db}
}

// GetBetaModeStatus 获取内测模式状态
// @Summary 获取内测模式状态
// @Description 获取当前系统的内测模式状态
// @Tags 内测模式
// @Accept json
// @Produce json
// @Success 200 {object} map[string]bool "返回内测模式状态"
// @Router /beta/status [get]
func (bc *BetaModeController) GetBetaModeStatus(c *gin.Context) {
	cfg := config.GetConfig()
	c.JSON(http.StatusOK, gin.H{
		"is_beta_mode": cfg.IsBetaMode,
	})
}

// ToggleBetaMode 切换内测模式状态
// @Summary 切换内测模式状态
// @Description 开启或关闭系统的内测模式（仅管理员可用）
// @Tags 内测模式
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body object true "请求参数" SchemaExample({"enabled": true})
// @Success 200 {object} map[string]interface{} "返回操作结果"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 403 {object} map[string]string "权限不足"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /admin/beta/toggle [post]
func (bc *BetaModeController) ToggleBetaMode(c *gin.Context) {
	claims, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "未登录",
		})
		return
	}

	userClaims, ok := claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "无效的令牌信息",
		})
		return
	}

	userRole, ok := userClaims["role"].(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "无效的角色信息",
		})
		return
	}

	if userRole != models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "只有管理员可以切换内测模式",
		})
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("请求参数绑定失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("无效的请求参数: %v", err),
		})
		return
	}

	if err := config.SetBetaMode(req.Enabled); err != nil {
		log.Printf("设置内测模式失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("设置内测模式失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "内测模式状态已更新",
		"is_beta_mode": req.Enabled,
	})
}

// UpdateUserBetaAccess 更新用户的内测访问权限
// @Summary 更新用户的内测访问权限
// @Description 更新指定用户的内测版本访问权限（仅管理员可用）
// @Tags 内测模式
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body object true "请求参数" SchemaExample({"user_id": 1, "is_allowed": true})
// @Success 200 {object} map[string]string "返回操作结果"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 403 {object} map[string]string "权限不足"
// @Failure 404 {object} map[string]string "用户不存在"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /admin/beta/user-access [post]
func (bc *BetaModeController) UpdateUserBetaAccess(c *gin.Context) {
	claims, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "未登录",
		})
		return
	}

	userClaims, ok := claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "无效的令牌信息",
		})
		return
	}

	userRole, ok := userClaims["role"].(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "无效的角色信息",
		})
		return
	}

	if userRole != models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "只有管理员可以更新用户的内测访问权限",
		})
		return
	}

	var req struct {
		UserID    uint `json:"user_id" binding:"required"`
		IsAllowed bool `json:"is_allowed"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("请求参数绑定失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("无效的请求参数: %v", err),
		})
		return
	}

	var targetUser models.User
	if err := bc.db.First(&targetUser, req.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "用户不存在",
		})
		return
	}

	targetUser.IsAllowed = req.IsAllowed
	if err := bc.db.Save(&targetUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "更新用户权限失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "用户内测访问权限已更新",
	})
}
