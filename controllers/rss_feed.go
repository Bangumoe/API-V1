package controllers

import (
	"fmt"
	"net/http"

	"backend/models"
	"backend/services/rss"
	"backend/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RSSResponse 定义通用响应结构
type RSSResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// @Summary 获取所有RSS订阅源
// @Description 获取系统中所有已配置的RSS订阅源列表
// @Tags RSS订阅源管理
// @Produce json
// @Success 200 {object} RSSResponse{data=[]models.RSSFeedResponse}
// @Failure 500 {object} RSSResponse
// @Router /rss_feeds [get]
func GetAllRSSFeeds(c *gin.Context) {
	var feeds []models.RSSFeed
	if err := models.DB.Find(&feeds).Error; err != nil {
		utils.LogError("获取RSS订阅源列表失败", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "获取RSS订阅源列表失败", "error": err.Error()})
		return
	}

	response := make([]models.RSSFeedResponse, len(feeds))
	for i, feed := range feeds {
		response[i] = models.RSSFeedResponse{
			ID:             feed.ID,
			Name:           feed.Name,
			URL:            feed.URL,
			UpdateInterval: feed.UpdateInterval,
			Keywords:       feed.Keywords,
			Priority:       feed.Priority,
			ParserType:     feed.ParserType,
			CreatedAt:      feed.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:      feed.UpdatedAt.Format("2006-01-02 15:04:05"),
			PageStart:      feed.PageStart,
			PageEnd:        feed.PageEnd,
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "获取RSS订阅源列表成功", "data": response})
}

// @Summary 获取单个RSS订阅源
// @Description 根据ID获取特定的RSS订阅源详细信息
// @Tags RSS订阅源管理
// @Produce json
// @Param id path int true "RSS订阅源ID"
// @Success 200 {object} RSSResponse{data=models.RSSFeedResponse}
// @Failure 404 {object} RSSResponse
// @Failure 500 {object} RSSResponse
// @Router /rss_feeds/{id} [get]
func GetRSSFeedByID(c *gin.Context) {
	id := c.Param("id")
	var feed models.RSSFeed
	if err := models.DB.First(&feed, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": http.StatusNotFound, "message": fmt.Sprintf("ID为%s的RSS订阅源不存在", id)})
		} else {
			utils.LogError(fmt.Sprintf("获取ID为%s的RSS订阅源失败", id), err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "获取RSS订阅源失败", "error": err.Error()})
		}
		return
	}

	response := models.RSSFeedResponse{
		ID:             feed.ID,
		Name:           feed.Name,
		URL:            feed.URL,
		UpdateInterval: feed.UpdateInterval,
		Keywords:       feed.Keywords,
		Priority:       feed.Priority,
		ParserType:     feed.ParserType,
		CreatedAt:      feed.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:      feed.UpdatedAt.Format("2006-01-02 15:04:05"),
		PageStart:      feed.PageStart,
		PageEnd:        feed.PageEnd,
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "获取RSS订阅源成功", "data": response})
}

// @Summary 创建RSS订阅源
// @Description 创建新的RSS订阅源
// @Tags RSS订阅源管理
// @Accept json
// @Produce json
// @Param feed body models.RSSFeedRequest true "RSS订阅源信息"
// @Success 201 {object} RSSResponse{data=models.RSSFeedResponse}
// @Failure 400 {object} RSSResponse
// @Failure 500 {object} RSSResponse
// @Router /rss_feeds [post]
func CreateRSSFeed(c *gin.Context) {
	var req models.RSSFeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "请求参数无效", "error": err.Error()})
		return
	}

	// 检查URL是否已存在
	var existingFeed models.RSSFeed
	err := models.DB.Where("url = ?", req.URL).First(&existingFeed).Error
	if err == nil {
		// 查到记录，说明已存在
		c.JSON(http.StatusConflict, gin.H{"code": http.StatusConflict, "message": "该URL已存在"})
		return
	} else if err != gorm.ErrRecordNotFound {
		// 其他数据库错误
		utils.LogError("检查RSS订阅源URL失败", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "检查RSS订阅源URL失败", "error": err.Error()})
		return
	}

	feed := models.RSSFeed{
		Name:           req.Name,
		URL:            req.URL,
		UpdateInterval: req.UpdateInterval,
		Keywords:       req.Keywords,
		Priority:       req.Priority,
		ParserType:     req.ParserType,
		PageStart:      req.PageStart,
		PageEnd:        req.PageEnd,
	}

	if err := models.DB.Create(&feed).Error; err != nil {
		utils.LogError("创建RSS订阅源失败", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "创建RSS订阅源失败", "error": err.Error()})
		return
	}

	response := models.RSSFeedResponse{
		ID:             feed.ID,
		Name:           feed.Name,
		URL:            feed.URL,
		UpdateInterval: feed.UpdateInterval,
		Keywords:       feed.Keywords,
		Priority:       feed.Priority,
		ParserType:     feed.ParserType,
		CreatedAt:      feed.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:      feed.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	c.JSON(http.StatusCreated, gin.H{"code": http.StatusCreated, "message": "创建RSS订阅源成功", "data": response})
}

// @Summary 更新RSS订阅源
// @Description 更新指定ID的RSS订阅源信息
// @Tags RSS订阅源管理
// @Accept json
// @Produce json
// @Param id path int true "RSS订阅源ID"
// @Param feed body models.RSSFeedRequest true "RSS订阅源信息"
// @Success 200 {object} RSSResponse{data=models.RSSFeedResponse}
// @Failure 400 {object} RSSResponse
// @Failure 404 {object} RSSResponse
// @Failure 500 {object} RSSResponse
// @Router /rss_feeds/{id} [put]
func UpdateRSSFeed(c *gin.Context) {
	id := c.Param("id")
	var feed models.RSSFeed
	if err := models.DB.First(&feed, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": http.StatusNotFound, "message": fmt.Sprintf("ID为%s的RSS订阅源不存在", id)})
		} else {
			utils.LogError(fmt.Sprintf("获取ID为%s的RSS订阅源失败", id), err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "更新RSS订阅源失败", "error": err.Error()})
		}
		return
	}

	var req models.RSSFeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "请求参数无效", "error": err.Error()})
		return
	}

	feed.Name = req.Name
	feed.URL = req.URL
	feed.UpdateInterval = req.UpdateInterval
	feed.Keywords = req.Keywords
	feed.Priority = req.Priority
	feed.ParserType = req.ParserType
	feed.PageStart = req.PageStart
	feed.PageEnd = req.PageEnd

	// 从数据库中重新查询以确保数据是最新的
	if err := models.DB.Save(&feed).Error; err != nil {
		utils.LogError(fmt.Sprintf("更新ID为%s的RSS订阅源失败", id), err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "更新RSS订阅源失败", "error": err.Error()})
		return
	}

	// 重新从数据库获取完整的feed信息
	var updatedFeed models.RSSFeed
	if err := models.DB.First(&updatedFeed, id).Error; err != nil {
		utils.LogError(fmt.Sprintf("获取更新后的RSS订阅源失败: %s", id), err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "获取更新后的RSS订阅源失败", "error": err.Error()})
		return
	}

	response := models.RSSFeedResponse{
		ID:             updatedFeed.ID,
		Name:           updatedFeed.Name,
		URL:            updatedFeed.URL,
		UpdateInterval: updatedFeed.UpdateInterval,
		Keywords:       updatedFeed.Keywords,
		Priority:       updatedFeed.Priority,
		ParserType:     updatedFeed.ParserType,
		CreatedAt:      updatedFeed.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:      updatedFeed.UpdatedAt.Format("2006-01-02 15:04:05"),
		PageStart:      updatedFeed.PageStart,
		PageEnd:        updatedFeed.PageEnd,
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "更新RSS订阅源成功", "data": response})
}

// @Summary 手动更新所有RSS订阅
// @Description 手动触发RSS订阅源的更新任务，立即返回并在后台执行更新
// @Tags RSS订阅源管理
// @Produce json
// @Success 200 {object} RSSResponse
// @Router /rss_feeds/update [post]
func ManualUpdateRSSFeeds(c *gin.Context) {
	// 立即返回成功响应
	c.JSON(http.StatusOK, RSSResponse{
		Code:    http.StatusOK,
		Message: "RSS更新任务已在后台触发",
	})

	// 在后台执行更新操作
	go func() {
		if err := rss.UpdateRSSFeeds(models.DB, true); err != nil {
			utils.LogError("后台RSS更新失败", err)
		} else {
			utils.LogInfo("后台RSS更新完成")
		}
	}()
}

// @Summary 手动更新指定RSS订阅
// @Description 手动触发指定ID的RSS订阅源的更新任务
// @Tags RSS订阅源管理
// @Produce json
// @Param id path int true "RSS订阅源ID"
// @Success 200 {object} RSSResponse
// @Failure 404 {object} RSSResponse
// @Failure 500 {object} RSSResponse
// @Router /rss_feeds/{id}/update [post]
func UpdateRSSFeedByID(c *gin.Context) {
	id := c.Param("id")

	// 检查Feed是否存在
	var feed models.RSSFeed
	if err := models.DB.First(&feed, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    http.StatusNotFound,
				"message": fmt.Sprintf("ID为%s的RSS订阅源不存在", id),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "获取RSS订阅源失败",
			"error":   err.Error(),
		})
		return
	}

	// 立即返回响应
	c.JSON(http.StatusOK, RSSResponse{
		Code:    http.StatusOK,
		Message: fmt.Sprintf("RSS订阅源[ID:%s]更新任务已在后台触发", id),
	})

	// 在后台执行更新操作
	go func() {
		if err := rss.UpdateSingleRSSFeed(models.DB, feed.ID); err != nil {
			utils.LogError(fmt.Sprintf("后台更新RSS订阅源[ID:%s]失败", id), err)
		} else {
			utils.LogInfo(fmt.Sprintf("后台更新RSS订阅源[ID:%s]完成", id))
		}
	}()
}

// @Summary 删除RSS订阅源
// @Description 删除指定ID的RSS订阅源
// @Tags RSS订阅源管理
// @Produce json
// @Param id path int true "RSS订阅源ID"
// @Success 200 {object} RSSResponse
// @Failure 404 {object} RSSResponse
// @Failure 500 {object} RSSResponse
// @Router /rss_feeds/{id} [delete]
func DeleteRSSFeed(c *gin.Context) {
	id := c.Param("id")
	var feed models.RSSFeed
	if err := models.DB.First(&feed, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": http.StatusNotFound, "message": fmt.Sprintf("ID为%s的RSS订阅源不存在", id)})
		} else {
			utils.LogError(fmt.Sprintf("获取ID为%s的RSS订阅源失败", id), err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "删除RSS订阅源失败", "error": err.Error()})
		}
		return
	}

	if err := models.DB.Unscoped().Delete(&feed).Error; err != nil {
		utils.LogError(fmt.Sprintf("删除ID为%s的RSS订阅源失败", id), err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "删除RSS订阅源失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "删除RSS订阅源成功"})
}
