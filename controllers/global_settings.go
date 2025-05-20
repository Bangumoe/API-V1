package controllers

import (
	"backend/models"
	"backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GlobalSettingsUpdateRequest 用于更新全局设置的请求体
type GlobalSettingsUpdateRequest struct {
	GlobalKeywords    string `json:"global_keywords" example:"动画,动漫" description:"全局关键词"`
	ExcludeKeywords   string `json:"exclude_keywords" example:"预告,PV" description:"全局排除关键词"`
	SubGroupBlacklist string `json:"sub_group_blacklist" example:"字幕组1,字幕组2" description:"字幕组黑名单"`
}

// GlobalSettingsResponse 用于Swagger文档的全局设置响应模型
type GlobalSettingsResponse struct {
	ID                uint   `json:"id" example:"1" description:"设置ID"`
	GlobalKeywords    string `json:"global_keywords" example:"动画,动漫" description:"全局关键词"`
	ExcludeKeywords   string `json:"exclude_keywords" example:"预告,PV" description:"全局排除关键词"`
	SubGroupBlacklist string `json:"sub_group_blacklist" example:"字幕组1,字幕组2" description:"字幕组黑名单"`
	CreatedAt         string `json:"created_at" example:"2024-05-20T12:00:00Z" description:"创建时间"`
	UpdatedAt         string `json:"updated_at" example:"2024-05-20T12:00:00Z" description:"更新时间"`
}

// @Summary 获取全局设置
// @Description 获取全局关键词、排除关键词和字幕组黑名单设置
// @Tags 全局设置
// @Produce json
// @Success 200 {object} GlobalSettingsResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/settings [get]
func GetGlobalSettings(c *gin.Context) {
	settings, err := models.GetGlobalSettings()
	if err != nil {
		utils.LogError("获取全局设置失败", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "获取全局设置失败",
			"error":   err.Error(),
		})
		return
	}

	response := GlobalSettingsResponse{
		ID:                settings.ID,
		GlobalKeywords:    settings.GlobalKeywords,
		ExcludeKeywords:   settings.ExcludeKeywords,
		SubGroupBlacklist: settings.SubGroupBlacklist,
		CreatedAt:         settings.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:         settings.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "获取全局设置成功",
		"data":    response,
	})
}

// @Summary 更新全局设置
// @Description 更新全局关键词、排除关键词和字幕组黑名单设置
// @Tags 全局设置
// @Accept json
// @Produce json
// @Param settings body GlobalSettingsUpdateRequest true "全局设置"
// @Success 200 {object} GlobalSettingsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/settings [put]
func UpdateGlobalSettings(c *gin.Context) {
	var req GlobalSettingsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "请求参数无效",
			"error":   err.Error(),
		})
		return
	}

	settings, err := models.GetGlobalSettings()
	if err != nil {
		utils.LogError("获取全局设置失败", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "获取全局设置失败",
			"error":   err.Error(),
		})
		return
	}

	settings.GlobalKeywords = req.GlobalKeywords
	settings.ExcludeKeywords = req.ExcludeKeywords
	settings.SubGroupBlacklist = req.SubGroupBlacklist

	if err := models.UpdateGlobalSettings(settings); err != nil {
		utils.LogError("更新全局设置失败", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "更新全局设置失败",
			"error":   err.Error(),
		})
		return
	}

	response := GlobalSettingsResponse{
		ID:                settings.ID,
		GlobalKeywords:    settings.GlobalKeywords,
		ExcludeKeywords:   settings.ExcludeKeywords,
		SubGroupBlacklist: settings.SubGroupBlacklist,
		CreatedAt:         settings.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:         settings.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "更新全局设置成功",
		"data":    response,
	})
}
