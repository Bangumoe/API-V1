package controllers

import (
	"backend/models"
	"backend/utils"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// BangumiResponse 定义通用响应结构
type BangumiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Total   int64       `json:"total,omitempty"`
}

// BangumiSearchParams 定义搜索参数
type BangumiSearchParams struct {
	Title    string `form:"title"`     // 标题搜索
	Year     string `form:"year"`      // 年份筛选
	Season   int    `form:"season"`    // 季度筛选
	Source   string `form:"source"`    // 来源筛选
	Page     int    `form:"page"`      // 页码
	PageSize int    `form:"page_size"` // 每页数量
}

// RSSItemSearchParams 定义RSS条目搜索参数
type RSSItemSearchParams struct {
	Group    string   `form:"group"`     // 字幕组筛选
	Source   string   `form:"source"`    // 来源筛选
	MinEp    *float64 `form:"min_ep"`    // 最小集数
	MaxEp    *float64 `form:"max_ep"`    // 最大集数
	Episode  *float64 `form:"episode"`   // 特定集数
	Page     int      `form:"page"`      // 页码
	PageSize int      `form:"page_size"` // 每页数量
}

// GroupedRSSItems 字幕组分类的RSS条目
type GroupedRSSItems struct {
	GroupName string      `json:"group_name"`
	Episodes  []RSSDetail `json:"episodes"`
}

// RSSDetail RSS条目详细信息
type RSSDetail struct {
	Episode     *float64 `json:"episode"`
	Resolution  string   `json:"resolution"`
	URL         string   `json:"url"`
	ReleaseDate string   `json:"release_date"`
}

// @Summary 获取所有番剧
// @Description 获取系统中所有番剧列表，支持分页
// @Tags 番剧管理
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} BangumiResponse
// @Failure 500 {object} BangumiResponse
// @Router /bangumi [get]
func GetAllBangumi(c *gin.Context) {
	var params BangumiSearchParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, BangumiResponse{
			Code:    http.StatusBadRequest,
			Message: "无效的请求参数",
			Error:   err.Error(),
		})
		return
	}

	// 设置默认分页参数
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}

	var bangumis []models.Bangumi
	var total int64

	// 构建查询
	query := models.DB.Model(&models.Bangumi{})

	// 执行计数
	if err := query.Count(&total).Error; err != nil {
		utils.LogError("获取番剧总数失败", err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "获取番剧列表失败",
			Error:   err.Error(),
		})
		return
	}

	// 执行分页查询
	offset := (params.Page - 1) * params.PageSize
	if err := query.Offset(offset).Limit(params.PageSize).Find(&bangumis).Error; err != nil {
		utils.LogError("获取番剧列表失败", err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "获取番剧列表失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, BangumiResponse{
		Code:    http.StatusOK,
		Message: "获取番剧列表成功",
		Data:    bangumis,
		Total:   total,
	})
}

// @Summary 搜索番剧
// @Description 根据条件搜索番剧
// @Tags 番剧管理
// @Produce json
// @Param title query string false "标题关键词"
// @Param year query string false "年份"
// @Param season query int false "季度"
// @Param source query string false "来源"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} BangumiResponse
// @Failure 400 {object} BangumiResponse
// @Failure 500 {object} BangumiResponse
// @Router /bangumi/search [get]
func SearchBangumi(c *gin.Context) {
	var params BangumiSearchParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, BangumiResponse{
			Code:    http.StatusBadRequest,
			Message: "无效的请求参数",
			Error:   err.Error(),
		})
		return
	}

	// 设置默认分页参数
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}

	var bangumis []models.Bangumi
	var total int64

	// 构建查询
	query := models.DB.Model(&models.Bangumi{})

	// 添加搜索条件
	if params.Title != "" {
		query = query.Where("official_title LIKE ?", "%"+params.Title+"%")
	}
	if params.Year != "" {
		query = query.Where("year = ?", params.Year)
	}
	if params.Season > 0 {
		query = query.Where("season = ?", params.Season)
	}
	if params.Source != "" {
		query = query.Where("source = ?", params.Source)
	}

	// 执行计数
	if err := query.Count(&total).Error; err != nil {
		utils.LogError("获取搜索结果总数失败", err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "搜索番剧失败",
			Error:   err.Error(),
		})
		return
	}

	// 执行分页查询
	offset := (params.Page - 1) * params.PageSize
	if err := query.Offset(offset).Limit(params.PageSize).Find(&bangumis).Error; err != nil {
		utils.LogError("获取搜索结果失败", err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "搜索番剧失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, BangumiResponse{
		Code:    http.StatusOK,
		Message: "搜索番剧成功",
		Data:    bangumis,
		Total:   total,
	})
}

// @Summary 获取番剧详情
// @Description 通过ID获取特定番剧的详细信息
// @Tags 番剧管理
// @Produce json
// @Param id path int true "番剧ID"
// @Success 200 {object} BangumiResponse
// @Failure 404 {object} BangumiResponse
// @Failure 500 {object} BangumiResponse
// @Router /bangumi/{id} [get]
func GetBangumiByID(c *gin.Context) {
	id := c.Param("id")

	var bangumi models.Bangumi
	if err := models.DB.First(&bangumi, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, BangumiResponse{
				Code:    http.StatusNotFound,
				Message: fmt.Sprintf("ID为%s的番剧不存在", id),
			})
		} else {
			utils.LogError(fmt.Sprintf("获取ID为%s的番剧失败", id), err)
			c.JSON(http.StatusInternalServerError, BangumiResponse{
				Code:    http.StatusInternalServerError,
				Message: "获取番剧详情失败",
				Error:   err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, BangumiResponse{
		Code:    http.StatusOK,
		Message: "获取番剧详情成功",
		Data:    bangumi,
	})
}

// @Summary 获取番剧关联的RSS条目
// @Description 通过番剧ID获取相关的RSS条目，支持字幕组和来源筛选，以及集数范围筛选
// @Tags 番剧管理
// @Produce json
// @Param id path int true "番剧ID"
// @Param group query string false "字幕组"
// @Param source query string false "来源"
// @Param min_ep query number false "最小集数"
// @Param max_ep query number false "最大集数"
// @Param episode query number false "特定集数"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} BangumiResponse
// @Failure 400 {object} BangumiResponse
// @Failure 404 {object} BangumiResponse
// @Failure 500 {object} BangumiResponse
// @Router /bangumi/items/{id} [get]
func GetBangumiRSSItems(c *gin.Context) {
	id := c.Param("id")

	// 验证番剧是否存在
	var bangumi models.Bangumi
	if err := models.DB.First(&bangumi, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, BangumiResponse{
				Code:    http.StatusNotFound,
				Message: fmt.Sprintf("ID为%s的番剧不存在", id),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "查询番剧失败",
			Error:   err.Error(),
		})
		return
	}

	var params RSSItemSearchParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, BangumiResponse{
			Code:    http.StatusBadRequest,
			Message: "无效的请求参数",
			Error:   err.Error(),
		})
		return
	}

	// 设置默认分页参数
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}

	// 构建查询，启用调试模式
	query := models.DB.Debug().Model(&models.RSSItem{}).
		Preload("Bangumi"). // 预加载番剧信息
		Preload("RssFeed"). // 预加载RSS源信息
		Where("bangumi_id = ?", id)

	// 添加表名
	query = query.Table("rss_items")

	// 添加筛选条件，只有当参数有实际值时才添加条件
	if params.Group != "" {
		query = query.Where("`group` = ?", params.Group)
	}
	if params.Source != "" {
		query = query.Where("source = ?", params.Source)
	}

	// 优化集数筛选逻辑
	if params.Episode != nil && *params.Episode > 0 {
		// 只有当Episode有值且大于0时才使用
		query = query.Where("episode = ?", *params.Episode)
	} else {
		// 范围筛选
		if params.MinEp != nil && *params.MinEp > 0 {
			query = query.Where("episode >= ?", params.MinEp)
		}
		if params.MaxEp != nil && *params.MaxEp > 0 {
			query = query.Where("episode <= ?", params.MaxEp)
		}
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		utils.LogError("获取RSS条目总数失败", err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "获取RSS条目总数失败",
			Error:   err.Error(),
		})
		return
	}

	// 执行分页查询，包含预加载的关联数据
	var rssItems []models.RSSItem
	offset := (params.Page - 1) * params.PageSize
	if err := query.
		Order("episode ASC"). // 按集数升序排序
		Offset(offset).
		Limit(params.PageSize).
		Find(&rssItems).Error; err != nil {
		utils.LogError("查询RSS条目失败", err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "查询RSS条目失败",
			Error:   err.Error(),
		})
		return
	}

	// 添加日志
	utils.LogInfo(fmt.Sprintf("查询到 %d 条记录，总数: %d", len(rssItems), total))

	c.JSON(http.StatusOK, BangumiResponse{
		Code:    http.StatusOK,
		Message: "获取RSS条目成功",
		Data:    rssItems,
		Total:   total,
	})
}

// @Summary 获取按字幕组分类的番剧RSS条目
// @Description 获取指定番剧ID的所有RSS条目，并按字幕组分类
// @Tags 番剧管理
// @Produce json
// @Param id path int true "番剧ID"
// @Success 200 {object} BangumiResponse
// @Failure 404 {object} BangumiResponse
// @Failure 500 {object} BangumiResponse
// @Router /bangumi/grouped_items/{id} [get]
func GetGroupedBangumiRSSItems(c *gin.Context) {
	id := c.Param("id")

	// 验证番剧是否存在
	var bangumi models.Bangumi
	if err := models.DB.First(&bangumi, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, BangumiResponse{
				Code:    http.StatusNotFound,
				Message: fmt.Sprintf("ID为%s的番剧不存在", id),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "查询番剧失败",
			Error:   err.Error(),
		})
		return
	}

	// 获取所有相关RSS条目
	var rssItems []models.RSSItem
	if err := models.DB.Where("bangumi_id = ?", id).
		Order("`group` ASC, episode ASC").
		Find(&rssItems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "获取RSS条目失败",
			Error:   err.Error(),
		})
		return
	}

	// 按字幕组分类
	groupedItems := make(map[string][]RSSDetail)
	for _, item := range rssItems {
		detail := RSSDetail{
			Episode:     item.Episode,
			Resolution:  item.Resolution,
			URL:         item.URL,
			ReleaseDate: item.ReleaseDate,
		}
		groupedItems[item.Group] = append(groupedItems[item.Group], detail)
	}

	// 转换为响应格式
	var result []GroupedRSSItems
	for groupName, episodes := range groupedItems {
		result = append(result, GroupedRSSItems{
			GroupName: groupName,
			Episodes:  episodes,
		})
	}

	// 按字幕组名称排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].GroupName < result[j].GroupName
	})

	c.JSON(http.StatusOK, BangumiResponse{
		Code:    http.StatusOK,
		Message: "获取分组RSS条目成功",
		Data:    result,
	})
}

// @Summary 获取番剧统计信息
// @Description 获取系统中番剧的统计信息
// @Tags 番剧管理
// @Produce json
// @Success 200 {object} BangumiResponse
// @Failure 500 {object} BangumiResponse
// @Router /bangumi/stats [get]
func GetBangumiStats(c *gin.Context) {
	var stats struct {
		Total       int64    `json:"total"`        // 总番剧数
		YearStats   []string `json:"year_stats"`   // 年份统计
		SourceStats []string `json:"source_stats"` // 来源统计
	}

	// 获取总数
	if err := models.DB.Model(&models.Bangumi{}).Count(&stats.Total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "获取统计信息失败",
			Error:   err.Error(),
		})
		return
	}

	// 获取年份统计
	if err := models.DB.Model(&models.Bangumi{}).
		Distinct("year").
		Where("year IS NOT NULL").
		Pluck("year", &stats.YearStats).Error; err != nil {
		utils.LogError("获取年份统计失败", err)
	}

	// 获取来源统计
	if err := models.DB.Model(&models.Bangumi{}).
		Distinct("source").
		Where("source IS NOT NULL").
		Pluck("source", &stats.SourceStats).Error; err != nil {
		utils.LogError("获取来源统计失败", err)
	}

	c.JSON(http.StatusOK, BangumiResponse{
		Code:    http.StatusOK,
		Message: "获取统计信息成功",
		Data:    stats,
	})
}

// @Summary 获取特定字幕组的特定集数信息
// @Description 获取指定番剧的特定字幕组的特定集数的详细信息
// @Tags 番剧管理
// @Produce json
// @Param id path int true "番剧ID"
// @Param group query string true "字幕组名称"
// @Param episode query number true "集数"
// @Success 200 {object} BangumiResponse
// @Failure 404 {object} BangumiResponse
// @Failure 500 {object} BangumiResponse
// @Router /bangumi/{id}/group_episode [get]
func GetGroupEpisodeInfo(c *gin.Context) {
	id := c.Param("id")
	group := c.Query("group")
	episodeStr := c.Query("episode")

	if group == "" || episodeStr == "" {
		c.JSON(http.StatusBadRequest, BangumiResponse{
			Code:    http.StatusBadRequest,
			Message: "字幕组和集数参数不能为空",
		})
		return
	}

	episode, err := strconv.ParseFloat(episodeStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, BangumiResponse{
			Code:    http.StatusBadRequest,
			Message: "无效的集数参数",
			Error:   err.Error(),
		})
		return
	}

	// 查询特定条目
	var rssItem models.RSSItem
	err = models.DB.Where("bangumi_id = ? AND `group` = ? AND episode = ?", id, group, episode).
		First(&rssItem).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, BangumiResponse{
				Code:    http.StatusNotFound,
				Message: fmt.Sprintf("未找到字幕组[%s]的第%.0f集资源", group, episode),
			})
		} else {
			utils.LogError("查询RSS条目失败", err)
			c.JSON(http.StatusInternalServerError, BangumiResponse{
				Code:    http.StatusInternalServerError,
				Message: "查询失败",
				Error:   err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, BangumiResponse{
		Code:    http.StatusOK,
		Message: "获取资源信息成功",
		Data: map[string]interface{}{
			"episode":      rssItem.Episode,
			"group":        rssItem.Group,
			"url":          rssItem.URL,
			"resolution":   rssItem.Resolution,
			"release_date": rssItem.ReleaseDate,
		},
	})
}
