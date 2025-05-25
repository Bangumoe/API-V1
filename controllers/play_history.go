package controllers

import (
	"backend/models"
	"backend/utils"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"
)

type HistoryRequest struct {
	Url string `json:"url" binding:"required" example:"https://mikanime.tv/Download/20120812/6cfa68ddda6972015edbc4a505357ed4d2275f77.torrent"`
}

type HistoryResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Total   int64       `json:"total,omitempty"`
}

type HistoryArray struct {
	Id            uint      `json:"id"`
	Title         string    `json:"title"`
	Cover         string    `json:"cover"`
	Year          string    `json:"year"`
	Season        int       `json:"season"`
	HistoryTime   time.Time `json:"history_time"`
	ViewCount     uint      `json:"view_count"`
	FavoriteCount uint      `json:"favorite_count"`
	Episode       float64   `json:"episode"`
	HistoryId     uint      `json:"history_id"`
}

// @Summary 增加或更新观看历史记录
// @Description 点击立即播放的时候记录,如果历史记录已存在则更新上次播放时间
// @Tags 播放记录
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        login body HistoryRequest true "历史记录请求信息"
// @Success 200 {object} BangumiResponse
// @Failure 500 {object} BangumiResponse
// @Router /history/play_history [post]
func AddOrUpdatePlayHistroy(c *gin.Context) {
	info := "增加或更新观看历史记录"
	var body HistoryRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, HistoryResponse{
			Code:    http.StatusBadRequest,
			Message: "无效的请求参数",
			Error:   err.Error(),
		})
		return
	}

	var check_result models.RSSItem
	// 构建查询
	query := models.DB.Model(&models.RSSItem{}).Where("url = ?", body.Url).Order("id asc")

	// 根据url种子查询其id
	if err := query.First(&check_result).Error; err != nil {
		DatabaseErrorHandlerD(c, "查询rss_items是否存在失败", info+"失败", err)
		return
	}

	var uid uint
	if err := GetUserId(&uid, c); err != nil {
		return
	}

	// 构建新增的对象
	newItem := models.PlayHistory{
		RssItemsId: check_result.ID,
		UserId:     uid,
	}
	err := models.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"updated_at", "deleted_at"}),
	}).Create(&newItem).Error

	if err != nil {
		DatabaseErrorHandlerD(c, "upsert play_history 数据库失败", info+"失败", err)
		return
	}

	c.JSON(http.StatusOK, HistoryResponse{
		Code:    http.StatusOK,
		Message: info + "成功",
	})
}

// @Summary 删除观看历史记录
// @Description 软删除 需要编写定时器清理过期的记录和已经被删除的记录
// @Tags 播放记录
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param id path int true "historyId"
// @Success 200 {object} BangumiResponse
// @Failure 500 {object} BangumiResponse
// @Router /history/{id}/play_history [delete]
func DeletePlayHistroy(c *gin.Context) {
	info := "删除观看历史记录"

	idStr := c.Param("id")
	historyId, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, BangumiResponse{
			Code:    http.StatusBadRequest,
			Message: "无效的 historyId",
			Error:   err.Error(),
		})
		return
	}

	var uid uint
	if err := GetUserId(&uid, c); err != nil {
		return
	}

	var result models.PlayHistory

	// 构建更新
	err = models.DB.Raw(`update play_history t set t.deleted_at = CURRENT_TIMESTAMP
		where t.user_id = ? and t.deleted_at is null
		and t.id = ?`, uid, historyId).Scan(&result).Error
	if err != nil {
		DatabaseErrorHandlerD(c, "update 数据库失败", info+"失败", err)
		return
	}

	c.JSON(http.StatusOK, HistoryResponse{
		Code:    http.StatusOK,
		Message: info + "成功",
	})
}

func GetUserId(uid *uint, c *gin.Context) error {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, BangumiResponse{
			Code:    http.StatusUnauthorized,
			Message: "用户未认证",
		})
		return errors.New("")
	}
	// 断言 user_id 为 float64 (JWT 标准)，然后转换为 uint
	userIDFloat, ok := userIDValue.(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, BangumiResponse{
			Code:    http.StatusUnauthorized,
			Message: "无效的用户ID格式",
		})
		return errors.New("")
	}
	*uid = uint(userIDFloat)
	return nil
}

func DatabaseErrorHandlerD(c *gin.Context, error_message string, return_message string, err error) {
	utils.LogError(error_message, err)
	c.JSON(http.StatusInternalServerError, HistoryResponse{
		Code:    http.StatusInternalServerError,
		Message: return_message,
		Error:   err.Error(),
	})
}

// @Summary      获取用户观看历史记录
// @Description  获取当前登录用户观看历史记录
// @Tags 播放记录
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        page query int false "页码，默认1"
// @Param        page_size query int false "每页数量，默认10"
// @Success      200  {object}  Response
// @Failure      401  {object}  Response
// @Failure      500  {object}  Response
// @Router       /history/play_history [get]
func GetPlayHistory(c *gin.Context) {
	info := "获取用户观看历史记录"
	var uid uint
	if err := GetUserId(&uid, c); err != nil {
		return
	}

	// 获取分页参数
	page := utils.GetPage(c)
	pageSize := utils.GetPageSize(c)

	// 查询用户观看历史记录
	var history []HistoryArray
	var total int64

	// 获取总数
	err := models.DB.Raw(`select count(*)
		from bangumi t inner join rss_items t2
		on t2.bangumi_id = t.id
		inner join play_history t3
		on t3.rss_items_id = t2.id
		where t3.user_id = ?
		and t3.deleted_at is null`, uid).Scan(&total).Error
	if err != nil {
		DatabaseErrorHandlerD(c, "select 历史数据 数据库失败", info+"失败", err)
		return
	}

	total_pages := (total + int64(pageSize) - 1) / int64(pageSize)
	if page > int(total_pages) {
		page = int(total_pages)
	}

	// 获取历史记录列表
	err = models.DB.Raw(`select t.id, t.official_title as "title", t.poster_link as "cover",
		t.`+"`year`"+`, t.season, t3.updated_at as "history_time", t.view_count, t.favorite_count, 
		t2.url, t2.episode, t3.id as history_id
		from bangumi t inner join rss_items t2
		on t2.bangumi_id = t.id
		inner join play_history t3
		on t3.rss_items_id = t2.id
		where t3.user_id = ?
		and t3.deleted_at is null 
		order by t3.updated_at desc limit ? offset ?`, uid, pageSize, (page-1)*pageSize).Scan(&history).Error
	if err != nil {
		DatabaseErrorHandlerD(c, "select 历史数据 数据库失败", info+"失败", err)
		return
	}

	clientIP := c.ClientIP()
	fmt.Printf("GetPlayHistory - Client IP: %s\n", clientIP) // 添加日志

	// 保证list为[]而不是null
	if history == nil {
		history = make([]HistoryArray, 0)
	}

	// 处理封面链接
	for i := range history {
		coverPtr := &history[i].Cover
		history[i].Cover = *utils.GetPrefixedURL(clientIP, coverPtr)
	}

	// 构建响应数据
	c.JSON(http.StatusOK, Response{
		Data: gin.H{
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": total_pages,
			"list":        history,
		},
	})
}
