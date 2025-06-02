package controllers

import (
	"backend/models"
	"backend/utils"
	"database/sql"
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

// EpisodeInfo represents the innermost details of an episode.
type EpisodeInfo struct {
	Episode     *float64 `json:"episode"`
	URL         string   `json:"url"`
	ReleaseDate string   `json:"release_date"`
}

// SubGroupedEpisodes represents episodes grouped by subtitle type.
type SubGroupedEpisodes struct {
	SubType  string        `json:"sub_type"`
	Episodes []EpisodeInfo `json:"episodes"`
}

// ResolutionGroupedSubs represents subtitle groups classified by resolution.
type ResolutionGroupedSubs struct {
	ResolutionName string               `json:"resolution_name"`
	SubGroups      []SubGroupedEpisodes `json:"sub_groups"`
}

// GroupedByResolutionAndSub represents RSS items grouped by group, then resolution, then sub.
type GroupedByResolutionAndSub struct {
	GroupName   string                  `json:"group_name"`
	Resolutions []ResolutionGroupedSubs `json:"resolutions"`
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

	// 添加按年份倒序排序
	query = query.Order("year DESC")

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

	clientIP := c.ClientIP()
	fmt.Printf("GetAllBangumi - Client IP: %s\n", clientIP) // 添加日志

	// 根据请求来源处理 PosterLink
	for i := range bangumis {
		bangumis[i].PosterLink = utils.GetPrefixedURL(clientIP, bangumis[i].PosterLink)
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

	clientIP := c.ClientIP()
	fmt.Printf("SearchBangumi - Client IP: %s\n", clientIP) // 添加日志

	// 根据请求来源处理 PosterLink
	for i := range bangumis {
		bangumis[i].PosterLink = utils.GetPrefixedURL(clientIP, bangumis[i].PosterLink)
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

	clientIP := c.ClientIP()
	fmt.Printf("GetBangumiByID - Client IP: %s\n", clientIP) // 添加日志

	// 根据请求来源处理 PosterLink
	bangumi.PosterLink = utils.GetPrefixedURL(clientIP, bangumi.PosterLink)

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

// @Summary 获取按字幕组、分辨率、字幕类型分类的番剧RSS条目
// @Description 获取指定番剧ID的所有RSS条目，并按字幕组、分辨率、字幕类型分类。
// @Tags 番剧管理
// @Produce json
// @Param id path int true "番剧ID"
// @Success 200 {object} BangumiResponse{data=[]GroupedByResolutionAndSub} "成功获取分组RSS条目"
// @Failure 404 {object} BangumiResponse "番剧未找到"
// @Failure 500 {object} BangumiResponse "服务器内部错误"
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
		Order("`group` ASC, resolution ASC, sub ASC, episode ASC").
		Find(&rssItems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "获取RSS条目失败",
			Error:   err.Error(),
		})
		return
	}

	// 按字幕组、分辨率、字幕类型分类
	groupedData := make(map[string]map[string]map[string][]EpisodeInfo)

	for _, item := range rssItems {
		group := item.Group
		if group == "" {
			group = "未知字幕组"
		}
		resolution := item.Resolution
		if resolution == "" {
			resolution = "未知分辨率"
		}
		sub := item.Sub
		if sub == "" {
			sub = "未知字幕"
		}

		if _, ok := groupedData[group]; !ok {
			groupedData[group] = make(map[string]map[string][]EpisodeInfo)
		}
		if _, ok := groupedData[group][resolution]; !ok {
			groupedData[group][resolution] = make(map[string][]EpisodeInfo)
		}

		episodeDetail := EpisodeInfo{
			Episode:     item.Episode,
			URL:         item.URL,
			ReleaseDate: item.ReleaseDate,
		}
		groupedData[group][resolution][sub] = append(groupedData[group][resolution][sub], episodeDetail)
	}

	var result []GroupedByResolutionAndSub
	var groupNames []string
	for gn := range groupedData {
		groupNames = append(groupNames, gn)
	}
	sort.Strings(groupNames)

	for _, groupName := range groupNames {
		resolutionsMap := groupedData[groupName]
		var resolutionGroupedSubsList []ResolutionGroupedSubs
		var resolutionNames []string
		for rn := range resolutionsMap {
			resolutionNames = append(resolutionNames, rn)
		}
		sort.Strings(resolutionNames)

		for _, resolutionName := range resolutionNames {
			subsMap := resolutionsMap[resolutionName]
			var subGroupedEpisodesList []SubGroupedEpisodes
			var subTypes []string
			for st := range subsMap {
				subTypes = append(subTypes, st)
			}
			sort.Strings(subTypes)

			for _, subType := range subTypes {
				episodes := subsMap[subType]
				subGroupedEpisodesList = append(subGroupedEpisodesList, SubGroupedEpisodes{
					SubType:  subType,
					Episodes: episodes,
				})
			}
			resolutionGroupedSubsList = append(resolutionGroupedSubsList, ResolutionGroupedSubs{
				ResolutionName: resolutionName,
				SubGroups:      subGroupedEpisodesList,
			})
		}
		result = append(result, GroupedByResolutionAndSub{
			GroupName:   groupName,
			Resolutions: resolutionGroupedSubsList,
		})
	}

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

// @Summary 增加番剧点击量
// @Description 为指定ID的番剧增加一次点击量
// @Tags 番剧统计
// @Produce json
// @Param id path int true "番剧ID"
// @Success 200 {object} BangumiResponse "点击量增加成功"
// @Failure 400 {object} BangumiResponse "无效的番剧ID"
// @Failure 404 {object} BangumiResponse "番剧未找到"
// @Failure 500 {object} BangumiResponse "服务器内部错误"
// @Router /bangumi/{id}/view [post]
func IncrementViewCount(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, BangumiResponse{
			Code:    http.StatusBadRequest,
			Message: "无效的番剧ID",
			Error:   err.Error(),
		})
		return
	}

	var bangumi models.Bangumi
	// 使用 GORM 的 乐观锁 功能来处理并发更新
	result := models.DB.Model(&bangumi).Where("id = ?", uint(id)).UpdateColumn("view_count", gorm.Expr("view_count + ?", 1))

	if result.Error != nil {
		utils.LogError(fmt.Sprintf("增加番剧[%d]点击量失败", id), result.Error)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "增加点击量失败",
			Error:   result.Error.Error(),
		})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, BangumiResponse{
			Code:    http.StatusNotFound,
			Message: "番剧未找到或更新失败",
		})
		return
	}

	c.JSON(http.StatusOK, BangumiResponse{
		Code:    http.StatusOK,
		Message: "点击量增加成功",
	})
}

// @Summary 收藏/取消收藏番剧
// @Description 用户收藏或取消收藏指定ID的番剧
// @Tags 番剧统计
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "番剧ID"
// @Success 200 {object} BangumiResponse "操作成功"
// @Failure 400 {object} BangumiResponse "无效的番剧ID"
// @Failure 401 {object} BangumiResponse "用户未认证"
// @Failure 404 {object} BangumiResponse "番剧未找到"
// @Failure 500 {object} BangumiResponse "服务器内部错误"
// @Router /bangumi/{id}/favorite [post]
func ToggleFavorite(c *gin.Context) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, BangumiResponse{
			Code:    http.StatusUnauthorized,
			Message: "用户未认证",
		})
		return
	}

	// 断言 user_id 为 float64 (JWT 标准)，然后转换为 uint
	userIDFloat, ok := userIDValue.(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, BangumiResponse{
			Code:    http.StatusUnauthorized,
			Message: "无效的用户ID格式",
		})
		return
	}
	uid := uint(userIDFloat)

	idStr := c.Param("id")
	bangumiID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, BangumiResponse{
			Code:    http.StatusBadRequest,
			Message: "无效的番剧ID",
			Error:   err.Error(),
		})
		return
	}

	// 检查番剧是否存在
	var bangumi models.Bangumi
	if err := models.DB.First(&bangumi, uint(bangumiID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, BangumiResponse{
				Code:    http.StatusNotFound,
				Message: "番剧未找到",
			})
		} else {
			utils.LogError(fmt.Sprintf("查找番剧[%d]失败", bangumiID), err)
			c.JSON(http.StatusInternalServerError, BangumiResponse{
				Code:    http.StatusInternalServerError,
				Message: "服务器内部错误",
				Error:   err.Error(),
			})
		}
		return
	}

	tx := models.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 检查是否已收藏
	var favorite models.BangumiFavorite
	err = tx.Unscoped().Where("user_id = ? AND bangumi_id = ?", uid, uint(bangumiID)).First(&favorite).Error

	if err == gorm.ErrRecordNotFound {
		// 添加收藏
		newFavorite := models.BangumiFavorite{
			UserID:    uid,
			BangumiID: uint(bangumiID),
		}
		if err := tx.Create(&newFavorite).Error; err != nil {
			tx.Rollback()
			utils.LogError(fmt.Sprintf("用户[%d]收藏番剧[%d]失败", uid, bangumiID), err)
			c.JSON(http.StatusInternalServerError, BangumiResponse{
				Code:    http.StatusInternalServerError,
				Message: "收藏失败",
				Error:   err.Error(),
			})
			return
		}
		// 更新番剧收藏计数
		if err := tx.Model(&models.Bangumi{}).Where("id = ?", uint(bangumiID)).UpdateColumn("favorite_count", gorm.Expr("favorite_count + ?", 1)).Error; err != nil {
			tx.Rollback()
			utils.LogError(fmt.Sprintf("更新番剧[%d]收藏数失败", bangumiID), err)
			c.JSON(http.StatusInternalServerError, BangumiResponse{
				Code:    http.StatusInternalServerError,
				Message: "更新收藏数失败",
				Error:   err.Error(),
			})
			return
		}
		if err := tx.Commit().Error; err != nil {
			utils.LogError("提交收藏事务失败", err)
			c.JSON(http.StatusInternalServerError, BangumiResponse{
				Code:    http.StatusInternalServerError,
				Message: "收藏操作失败",
				Error:   err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, BangumiResponse{
			Code:    http.StatusOK,
			Message: "收藏成功",
		})
	} else if err == nil {
		// 取消收藏
		if err := tx.Unscoped().Delete(&favorite).Error; err != nil {
			tx.Rollback()
			utils.LogError(fmt.Sprintf("用户[%d]取消收藏番剧[%d]失败", uid, bangumiID), err)
			c.JSON(http.StatusInternalServerError, BangumiResponse{
				Code:    http.StatusInternalServerError,
				Message: "取消收藏失败",
				Error:   err.Error(),
			})
			return
		}
		// 更新番剧收藏计数，确保不会出现负数
		if err := tx.Model(&models.Bangumi{}).
			Where("id = ? AND favorite_count > 0", uint(bangumiID)).
			UpdateColumn("favorite_count", gorm.Expr("favorite_count - ?", 1)).Error; err != nil {
			tx.Rollback()
			utils.LogError(fmt.Sprintf("更新番剧[%d]收藏数失败", bangumiID), err)
			c.JSON(http.StatusInternalServerError, BangumiResponse{
				Code:    http.StatusInternalServerError,
				Message: "更新收藏数失败",
				Error:   err.Error(),
			})
			return
		}
		if err := tx.Commit().Error; err != nil {
			utils.LogError("提交取消收藏事务失败", err)
			c.JSON(http.StatusInternalServerError, BangumiResponse{
				Code:    http.StatusInternalServerError,
				Message: "取消收藏操作失败",
				Error:   err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, BangumiResponse{
			Code:    http.StatusOK,
			Message: "取消收藏成功",
		})
	} else {
		tx.Rollback()
		utils.LogError(fmt.Sprintf("查询用户[%d]对番剧[%d]的收藏状态失败", uid, bangumiID), err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "操作失败",
			Error:   err.Error(),
		})
	}
}

// @Summary 添加或更新番剧评分
// @Description 用户为指定ID的番剧添加或更新评分
// @Tags 番剧统计
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path int true "番剧ID"
// @Param rating body models.BangumiRatingRequest true "评分信息"
// @Success 200 {object} BangumiResponse{data=models.BangumiRatingResponse} "操作成功"
// @Failure 400 {object} BangumiResponse "无效的请求参数或番剧ID"
// @Failure 401 {object} BangumiResponse "用户未认证"
// @Failure 404 {object} BangumiResponse "番剧未找到"
// @Failure 500 {object} BangumiResponse "服务器内部错误"
// @Router /bangumi/{id}/rating [post]
func AddOrUpdateRating(c *gin.Context) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, BangumiResponse{
			Code:    http.StatusUnauthorized,
			Message: "用户未认证",
		})
		return
	}
	// 断言 user_id 为 float64 (JWT 标准)，然后转换为 uint
	userIDFloat, ok := userIDValue.(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, BangumiResponse{
			Code:    http.StatusUnauthorized,
			Message: "无效的用户ID格式",
		})
		return
	}
	uid := uint(userIDFloat)

	idStr := c.Param("id")
	bangumiID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, BangumiResponse{
			Code:    http.StatusBadRequest,
			Message: "无效的番剧ID",
			Error:   err.Error(),
		})
		return
	}

	var req models.BangumiRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, BangumiResponse{
			Code:    http.StatusBadRequest,
			Message: "无效的评分数据",
			Error:   err.Error(),
		})
		return
	}

	// 检查番剧是否存在
	var bangumi models.Bangumi
	if err := models.DB.First(&bangumi, uint(bangumiID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, BangumiResponse{
				Code:    http.StatusNotFound,
				Message: "番剧未找到",
			})
		} else {
			utils.LogError(fmt.Sprintf("查找番剧[%d]失败", bangumiID), err)
			c.JSON(http.StatusInternalServerError, BangumiResponse{
				Code:    http.StatusInternalServerError,
				Message: "服务器内部错误",
				Error:   err.Error(),
			})
		}
		return
	}

	tx := models.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var rating models.BangumiRating
	err = tx.Unscoped().Where("user_id = ? AND bangumi_id = ?", uid, uint(bangumiID)).First(&rating).Error

	var isNewRating bool
	if err == gorm.ErrRecordNotFound {
		// 新增评分
		isNewRating = true
		rating = models.BangumiRating{
			UserID:    uid,
			BangumiID: uint(bangumiID),
			Score:     req.Score,
			Comment:   req.Comment,
		}
		if err := tx.Create(&rating).Error; err != nil {
			tx.Rollback()
			utils.LogError(fmt.Sprintf("用户[%d]为番剧[%d]添加评分失败", uid, bangumiID), err)
			c.JSON(http.StatusInternalServerError, BangumiResponse{
				Code:    http.StatusInternalServerError,
				Message: "添加评分失败",
				Error:   err.Error(),
			})
			return
		}
	} else if err == nil {
		// 更新评分
		isNewRating = false
		rating.Score = req.Score
		rating.Comment = req.Comment
		if err := tx.Save(&rating).Error; err != nil {
			tx.Rollback()
			utils.LogError(fmt.Sprintf("用户[%d]更新番剧[%d]评分失败", uid, bangumiID), err)
			c.JSON(http.StatusInternalServerError, BangumiResponse{
				Code:    http.StatusInternalServerError,
				Message: "更新评分失败",
				Error:   err.Error(),
			})
			return
		}
	} else {
		tx.Rollback()
		utils.LogError(fmt.Sprintf("查询用户[%d]对番剧[%d]的评分失败", uid, bangumiID), err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "评分操作失败",
			Error:   err.Error(),
		})
		return
	}

	// 更新番剧的平均分和评分人数
	if err := updateBangumiRatingStats(tx, uint(bangumiID), isNewRating); err != nil {
		tx.Rollback()
		utils.LogError(fmt.Sprintf("更新番剧[%d]评分统计失败", bangumiID), err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "更新番剧统计信息失败",
			Error:   err.Error(),
		})
		return
	}

	if err := tx.Commit().Error; err != nil {
		utils.LogError("提交评分事务失败", err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "评分操作失败",
			Error:   err.Error(),
		})
		return
	}

	response := models.BangumiRatingResponse{
		ID:        rating.ID,
		UserID:    rating.UserID,
		BangumiID: rating.BangumiID,
		Score:     rating.Score,
		Comment:   rating.Comment,
		CreatedAt: rating.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: rating.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	c.JSON(http.StatusOK, BangumiResponse{
		Code:    http.StatusOK,
		Message: "评分操作成功",
		Data:    response,
	})
}

// updateBangumiRatingStats 更新番剧的平均分和评分人数
func updateBangumiRatingStats(tx *gorm.DB, bangumiID uint, isNewRating bool) error {
	var avgScore sql.NullFloat64
	var ratingCount int64

	// 计算总分和评分人数
	result := tx.Model(&models.BangumiRating{}).
		Where("bangumi_id = ?", bangumiID).
		Select("ROUND(AVG(score), 2) as avg_score, COUNT(*) as rating_count").
		Row().
		Scan(&avgScore, &ratingCount)
	if result != nil {
		return fmt.Errorf("计算评分统计失败: %v", result)
	}

	// 确保平均分在有效范围内
	score := 0.0
	if avgScore.Valid {
		score = avgScore.Float64
		if score < 0 {
			score = 0
		} else if score > 10 {
			score = 10
		}
	}

	// 更新 Bangumi 表
	updateData := map[string]interface{}{
		"rating_avg":   score,
		"rating_count": ratingCount,
	}

	if err := tx.Model(&models.Bangumi{}).Where("id = ?", bangumiID).Updates(updateData).Error; err != nil {
		return fmt.Errorf("更新番剧评分统计失败: %v", err)
	}

	return nil
}

// @Summary 获取用户对番剧的评分
// @Description 获取当前登录用户对指定ID番剧的评分信息
// @Tags 番剧统计
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "番剧ID"
// @Success 200 {object} BangumiResponse{data=models.BangumiRatingResponse} "获取评分成功"
// @Failure 400 {object} BangumiResponse "无效的番剧ID"
// @Failure 401 {object} BangumiResponse "用户未认证"
// @Failure 404 {object} BangumiResponse "未找到评分记录或番剧"
// @Failure 500 {object} BangumiResponse "服务器内部错误"
// @Router /bangumi/{id}/rating [get]
func GetUserRating(c *gin.Context) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, BangumiResponse{
			Code:    http.StatusUnauthorized,
			Message: "用户未认证",
		})
		return
	}
	// 断言 user_id 为 float64 (JWT 标准)，然后转换为 uint
	userIDFloat, ok := userIDValue.(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, BangumiResponse{
			Code:    http.StatusUnauthorized,
			Message: "无效的用户ID格式",
		})
		return
	}
	uid := uint(userIDFloat)

	idStr := c.Param("id")
	bangumiID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, BangumiResponse{
			Code:    http.StatusBadRequest,
			Message: "无效的番剧ID",
			Error:   err.Error(),
		})
		return
	}

	var rating models.BangumiRating
	err = models.DB.Unscoped().Where("user_id = ? AND bangumi_id = ?", uid, uint(bangumiID)).First(&rating).Error

	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, BangumiResponse{
			Code:    http.StatusNotFound,
			Message: "未找到评分记录",
		})
		return
	} else if err != nil {
		utils.LogError(fmt.Sprintf("查询用户[%d]对番剧[%d]的评分失败", uid, bangumiID), err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "获取评分失败",
			Error:   err.Error(),
		})
		return
	}

	response := models.BangumiRatingResponse{
		ID:        rating.ID,
		UserID:    rating.UserID,
		BangumiID: rating.BangumiID,
		Score:     rating.Score,
		Comment:   rating.Comment,
		CreatedAt: rating.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: rating.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	c.JSON(http.StatusOK, BangumiResponse{
		Code:    http.StatusOK,
		Message: "获取评分成功",
		Data:    response,
	})
}

// @Summary 删除用户对番剧的评分
// @Description 删除当前登录用户对指定ID番剧的评分
// @Tags 番剧统计
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "番剧ID"
// @Success 200 {object} BangumiResponse "删除评分成功"
// @Failure 400 {object} BangumiResponse "无效的番剧ID"
// @Failure 401 {object} BangumiResponse "用户未认证"
// @Failure 404 {object} BangumiResponse "未找到评分记录或番剧"
// @Failure 500 {object} BangumiResponse "服务器内部错误"
// @Router /bangumi/{id}/rating [delete]
func DeleteUserRating(c *gin.Context) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, BangumiResponse{
			Code:    http.StatusUnauthorized,
			Message: "用户未认证",
		})
		return
	}
	// 断言 user_id 为 float64 (JWT 标准)，然后转换为 uint
	userIDFloat, ok := userIDValue.(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, BangumiResponse{
			Code:    http.StatusUnauthorized,
			Message: "无效的用户ID格式",
		})
		return
	}
	uid := uint(userIDFloat)

	idStr := c.Param("id")
	bangumiID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, BangumiResponse{
			Code:    http.StatusBadRequest,
			Message: "无效的番剧ID",
			Error:   err.Error(),
		})
		return
	}

	tx := models.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var rating models.BangumiRating
	err = tx.Unscoped().Where("user_id = ? AND bangumi_id = ?", uid, uint(bangumiID)).First(&rating).Error

	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, BangumiResponse{
			Code:    http.StatusNotFound,
			Message: "未找到评分记录",
		})
		return
	} else if err != nil {
		tx.Rollback()
		utils.LogError(fmt.Sprintf("查询用户[%d]对番剧[%d]的评分失败", uid, bangumiID), err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "删除评分失败",
			Error:   err.Error(),
		})
		return
	}

	// 删除评分记录
	if err := tx.Unscoped().Delete(&rating).Error; err != nil {
		tx.Rollback()
		utils.LogError(fmt.Sprintf("删除用户[%d]对番剧[%d]的评分失败", uid, bangumiID), err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "删除评分失败",
			Error:   err.Error(),
		})
		return
	}

	// 更新番剧的平均分和评分人数
	if err := updateBangumiRatingStats(tx, uint(bangumiID), false); err != nil { // isNewRating is false because we are deleting
		tx.Rollback()
		utils.LogError(fmt.Sprintf("更新番剧[%d]评分统计失败", bangumiID), err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "更新番剧统计信息失败",
			Error:   err.Error(),
		})
		return
	}

	if err := tx.Commit().Error; err != nil {
		utils.LogError("提交删除评分事务失败", err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "删除评分操作失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, BangumiResponse{
		Code:    http.StatusOK,
		Message: "删除评分成功",
	})
}

// @Summary 获取番剧点击量统计
// @Description 获取番剧的点击量统计信息
// @Tags 番剧统计
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} BangumiResponse
// @Failure 500 {object} BangumiResponse
// @Router /bangumi/stats/views [get]
func GetBangumiViewStats(c *gin.Context) {
	var params struct {
		Page     int `form:"page"`
		PageSize int `form:"page_size"`
	}

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
	query := models.DB.Model(&models.Bangumi{}).Order("view_count DESC")

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

	// Update PosterLink for each bangumi
	clientIP := c.ClientIP()
	for i := range bangumis {
		bangumis[i].PosterLink = utils.GetPrefixedURL(clientIP, bangumis[i].PosterLink)
	}

	c.JSON(http.StatusOK, BangumiResponse{
		Code:    http.StatusOK,
		Message: "获取番剧点击量统计成功",
		Data:    bangumis,
		Total:   total,
	})
}

// @Summary 获取番剧收藏量统计
// @Description 获取番剧的收藏量统计信息
// @Tags 番剧统计
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} BangumiResponse
// @Failure 500 {object} BangumiResponse
// @Router /bangumi/stats/favorites [get]
func GetBangumiFavoriteStats(c *gin.Context) {
	var params struct {
		Page     int `form:"page"`
		PageSize int `form:"page_size"`
	}

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
	query := models.DB.Model(&models.Bangumi{}).Order("favorite_count DESC")

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

	// Update PosterLink for each bangumi
	clientIP := c.ClientIP()
	for i := range bangumis {
		bangumis[i].PosterLink = utils.GetPrefixedURL(clientIP, bangumis[i].PosterLink)
	}

	c.JSON(http.StatusOK, BangumiResponse{
		Code:    http.StatusOK,
		Message: "获取番剧收藏量统计成功",
		Data:    bangumis,
		Total:   total,
	})
}

// @Summary 获取番剧评分统计
// @Description 获取番剧的评分统计信息
// @Tags 番剧统计
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} BangumiResponse
// @Failure 500 {object} BangumiResponse
// @Router /bangumi/stats/ratings [get]
func GetBangumiRatingStats(c *gin.Context) {
	var params struct {
		Page     int `form:"page"`
		PageSize int `form:"page_size"`
	}

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

	// 构建查询，只查询有评分的番剧
	query := models.DB.Model(&models.Bangumi{}).
		Where("rating_count > 0").
		Order("rating_avg DESC, rating_count DESC")

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

	// Update PosterLink for each bangumi
	clientIP := c.ClientIP()
	for i := range bangumis {
		bangumis[i].PosterLink = utils.GetPrefixedURL(clientIP, bangumis[i].PosterLink)
	}

	c.JSON(http.StatusOK, BangumiResponse{
		Code:    http.StatusOK,
		Message: "获取番剧评分统计成功",
		Data:    bangumis,
		Total:   total,
	})
}

// @Summary 获取番剧综合排名
// @Description 获取番剧的综合排名（基于点击量、收藏量和评分的加权计算）
// @Tags 番剧统计
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} BangumiResponse
// @Failure 500 {object} BangumiResponse
// @Router /bangumi/stats/rankings [get]
func GetBangumiRankings(c *gin.Context) {
	var params struct {
		Page     int `form:"page"`
		PageSize int `form:"page_size"`
	}

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

	// 构建查询，计算综合得分
	// 综合得分 = 点击量 * 0.3 + 收藏量 * 0.3 + 评分 * 0.4
	query := models.DB.Model(&models.Bangumi{}).
		Select("*, (view_count * 0.3 + favorite_count * 0.3 + rating_avg * 0.4) as score").
		Order("score DESC")

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

	// Update PosterLink for each bangumi
	clientIP := c.ClientIP()
	for i := range bangumis {
		bangumis[i].PosterLink = utils.GetPrefixedURL(clientIP, bangumis[i].PosterLink)
	}

	c.JSON(http.StatusOK, BangumiResponse{
		Code:    http.StatusOK,
		Message: "获取番剧综合排名成功",
		Data:    bangumis,
		Total:   total,
	})
}

// @Summary 获取指定番剧的统计信息
// @Description 获取指定番剧的点击量、收藏量和评分统计信息
// @Tags 番剧统计
// @Produce json
// @Param id path int true "番剧ID"
// @Success 200 {object} BangumiResponse
// @Failure 404 {object} BangumiResponse
// @Failure 500 {object} BangumiResponse
// @Router /bangumi/{id}/stats [get]
func GetBangumiStatsByID(c *gin.Context) {
	idStr := c.Param("id")
	bangumiID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, BangumiResponse{
			Code:    http.StatusBadRequest,
			Message: "无效的番剧ID",
			Error:   err.Error(),
		})
		return
	}

	// 查询番剧信息
	var bangumi models.Bangumi
	if err := models.DB.First(&bangumi, uint(bangumiID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, BangumiResponse{
				Code:    http.StatusNotFound,
				Message: "番剧未找到",
			})
		} else {
			utils.LogError(fmt.Sprintf("获取番剧[%d]统计信息失败", bangumiID), err)
			c.JSON(http.StatusInternalServerError, BangumiResponse{
				Code:    http.StatusInternalServerError,
				Message: "获取番剧统计信息失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 构建统计信息
	stats := map[string]interface{}{
		"view_count":     bangumi.ViewCount,
		"favorite_count": bangumi.FavoriteCount,
		"rating_avg":     bangumi.RatingAvg,
		"rating_count":   bangumi.RatingCount,
		"is_favorite":    false, // 默认未收藏
	}

	// 获取用户ID并查询收藏状态
	if userIDValue, exists := c.Get("user_id"); exists {
		if userIDFloat, ok := userIDValue.(float64); ok {
			uid := uint(userIDFloat)
			var favorite models.BangumiFavorite
			if err := models.DB.Where("user_id = ? AND bangumi_id = ?", uid, uint(bangumiID)).First(&favorite).Error; err == nil {
				stats["is_favorite"] = true
			}
		}
	}

	c.JSON(http.StatusOK, BangumiResponse{
		Code:    http.StatusOK,
		Message: "获取番剧统计信息成功",
		Data:    stats,
	})
}

// @Summary 获取指定番剧的评分详情
// @Description 获取指定番剧的评分分布和详细统计信息
// @Tags 番剧统计
// @Produce json
// @Param id path int true "番剧ID"
// @Success 200 {object} BangumiResponse
// @Failure 404 {object} BangumiResponse
// @Failure 500 {object} BangumiResponse
// @Router /bangumi/{id}/rating_stats [get]
func GetBangumiRatingStatsByID(c *gin.Context) {
	idStr := c.Param("id")
	bangumiID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, BangumiResponse{
			Code:    http.StatusBadRequest,
			Message: "无效的番剧ID",
			Error:   err.Error(),
		})
		return
	}

	// 查询评分分布
	var ratingStats []struct {
		Score float64 `gorm:"column:score"`
		Count int64   `gorm:"column:count"`
	}

	if err := models.DB.Model(&models.BangumiRating{}).
		Select("score, COUNT(*) as count").
		Where("bangumi_id = ?", uint(bangumiID)).
		Group("score").
		Order("score ASC").
		Scan(&ratingStats).Error; err != nil {
		utils.LogError(fmt.Sprintf("获取番剧[%d]评分分布失败", bangumiID), err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "获取评分分布失败",
			Error:   err.Error(),
		})
		return
	}

	// 构建评分分布数据
	distribution := make(map[float64]int64)
	for _, stat := range ratingStats {
		distribution[stat.Score] = stat.Count
	}

	// 查询番剧基本信息
	var bangumi models.Bangumi
	if err := models.DB.First(&bangumi, uint(bangumiID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, BangumiResponse{
				Code:    http.StatusNotFound,
				Message: "番剧未找到",
			})
		} else {
			utils.LogError(fmt.Sprintf("获取番剧[%d]基本信息失败", bangumiID), err)
			c.JSON(http.StatusInternalServerError, BangumiResponse{
				Code:    http.StatusInternalServerError,
				Message: "获取番剧信息失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 构建完整响应
	response := map[string]interface{}{
		"rating_avg":   bangumi.RatingAvg,
		"rating_count": bangumi.RatingCount,
		"distribution": distribution,
	}

	c.JSON(http.StatusOK, BangumiResponse{
		Code:    http.StatusOK,
		Message: "获取评分统计信息成功",
		Data:    response,
	})
}

// @Summary 根据年份获取番剧列表
// @Description 获取指定年份的所有番剧，支持分页
// @Tags 番剧管理
// @Produce json
// @Param year path string true "年份"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} BangumiResponse
// @Failure 400 {object} BangumiResponse
// @Failure 500 {object} BangumiResponse
// @Router /bangumi/year/{year} [get]
func GetBangumiByYear(c *gin.Context) {
	year := c.Param("year")
	if year == "" {
		c.JSON(http.StatusBadRequest, BangumiResponse{
			Code:    http.StatusBadRequest,
			Message: "年份不能为空",
		})
		return
	}

	var params struct {
		Page     int `form:"page"`
		PageSize int `form:"page_size"`
	}

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
	query := models.DB.Model(&models.Bangumi{}).Where("year = ?", year)

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

	clientIP := c.ClientIP()
	fmt.Printf("SearchBangumi - Client IP: %s\n", clientIP) // 添加日志

	// 根据请求来源处理 PosterLink
	for i := range bangumis {
		bangumis[i].PosterLink = utils.GetPrefixedURL(clientIP, bangumis[i].PosterLink)
	}

	c.JSON(http.StatusOK, BangumiResponse{
		Code:    http.StatusOK,
		Message: "获取番剧列表成功",
		Data:    bangumis,
		Total:   total,
	})
}

// @Summary 获取所有番剧年份
// @Description 获取数据库中所有番剧的年份列表，按年份降序排列
// @Tags 番剧管理
// @Produce json
// @Success 200 {object} BangumiResponse
// @Failure 500 {object} BangumiResponse
// @Router /bangumi/years [get]
func GetBangumiYears(c *gin.Context) {
	var years []string

	// 查询所有不重复的年份，并按降序排列
	if err := models.DB.Model(&models.Bangumi{}).
		Distinct("year").
		Where("year IS NOT NULL AND year != ''").
		Order("year DESC").
		Pluck("year", &years).Error; err != nil {
		utils.LogError("获取番剧年份列表失败", err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "获取番剧年份列表失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, BangumiResponse{
		Code:    http.StatusOK,
		Message: "获取番剧年份列表成功",
		Data:    years,
	})
}

// @Summary 删除番剧
// @Description 根据ID删除指定的番剧（硬删除）
// @Tags 番剧管理
// @Produce json
// @Param id path int true "番剧ID"
// @Success 200 {object} BangumiResponse "删除成功"
// @Failure 400 {object} BangumiResponse "无效的番剧ID"
// @Failure 404 {object} BangumiResponse "番剧未找到"
// @Failure 500 {object} BangumiResponse "服务器内部错误"
// @Router /admin/bangumi/{id} [delete]
func DeleteBangumi(c *gin.Context) {
	idStr := c.Param("id")
	bangumiID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, BangumiResponse{
			Code:    http.StatusBadRequest,
			Message: "无效的番剧ID",
			Error:   err.Error(),
		})
		return
	}

	// 开始事务
	tx := models.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 检查番剧是否存在
	var bangumi models.Bangumi
	if err := tx.First(&bangumi, uint(bangumiID)).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, BangumiResponse{
				Code:    http.StatusNotFound,
				Message: "番剧未找到",
			})
		} else {
			utils.LogError(fmt.Sprintf("查找番剧[%d]失败", bangumiID), err)
			c.JSON(http.StatusInternalServerError, BangumiResponse{
				Code:    http.StatusInternalServerError,
				Message: "删除番剧失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 删除相关的评分记录
	if err := tx.Unscoped().Where("bangumi_id = ?", uint(bangumiID)).Delete(&models.BangumiRating{}).Error; err != nil {
		tx.Rollback()
		utils.LogError(fmt.Sprintf("删除番剧[%d]评分记录失败", bangumiID), err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "删除番剧评分记录失败",
			Error:   err.Error(),
		})
		return
	}

	// 删除相关的收藏记录
	if err := tx.Unscoped().Where("bangumi_id = ?", uint(bangumiID)).Delete(&models.BangumiFavorite{}).Error; err != nil {
		tx.Rollback()
		utils.LogError(fmt.Sprintf("删除番剧[%d]收藏记录失败", bangumiID), err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "删除番剧收藏记录失败",
			Error:   err.Error(),
		})
		return
	}

	// 删除相关的RSS条目
	if err := tx.Unscoped().Where("bangumi_id = ?", uint(bangumiID)).Delete(&models.RSSItem{}).Error; err != nil {
		tx.Rollback()
		utils.LogError(fmt.Sprintf("删除番剧[%d]RSS条目失败", bangumiID), err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "删除番剧RSS条目失败",
			Error:   err.Error(),
		})
		return
	}

	// 硬删除番剧
	if err := tx.Unscoped().Delete(&bangumi).Error; err != nil {
		tx.Rollback()
		utils.LogError(fmt.Sprintf("删除番剧[%d]失败", bangumiID), err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "删除番剧失败",
			Error:   err.Error(),
		})
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		utils.LogError(fmt.Sprintf("提交删除番剧[%d]事务失败", bangumiID), err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "删除番剧失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, BangumiResponse{
		Code:    http.StatusOK,
		Message: "删除番剧成功",
	})
}

// @Summary 更新番剧信息
// @Description 根据ID更新指定番剧的信息
// @Tags 番剧管理
// @Accept json
// @Produce json
// @Param id path int true "番剧ID"
// @Param bangumi body models.BangumiUpdateRequest true "番剧更新信息"
// @Success 200 {object} BangumiResponse "更新成功"
// @Failure 400 {object} BangumiResponse "无效的请求参数"
// @Failure 404 {object} BangumiResponse "番剧未找到"
// @Failure 500 {object} BangumiResponse "服务器内部错误"
// @Router /admin/bangumi/{id} [put]
func UpdateBangumi(c *gin.Context) {
	idStr := c.Param("id")
	bangumiID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, BangumiResponse{
			Code:    http.StatusBadRequest,
			Message: "无效的番剧ID",
			Error:   err.Error(),
		})
		return
	}

	var req models.BangumiUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, BangumiResponse{
			Code:    http.StatusBadRequest,
			Message: "无效的请求参数",
			Error:   err.Error(),
		})
		return
	}

	// 检查番剧是否存在
	var bangumi models.Bangumi
	if err := models.DB.First(&bangumi, uint(bangumiID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, BangumiResponse{
				Code:    http.StatusNotFound,
				Message: "番剧未找到",
			})
		} else {
			utils.LogError(fmt.Sprintf("查找番剧[%d]失败", bangumiID), err)
			c.JSON(http.StatusInternalServerError, BangumiResponse{
				Code:    http.StatusInternalServerError,
				Message: "更新番剧失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 更新番剧信息
	updates := map[string]interface{}{}
	if req.OfficialTitle != "" {
		updates["official_title"] = req.OfficialTitle
	}
	if req.Year != nil {
		updates["year"] = req.Year
	}
	if req.Season > 0 {
		updates["season"] = req.Season
	}
	if req.PosterLink != nil {
		updates["poster_link"] = req.PosterLink
	}

	if err := models.DB.Model(&bangumi).Updates(updates).Error; err != nil {
		utils.LogError(fmt.Sprintf("更新番剧[%d]失败", bangumiID), err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "更新番剧失败",
			Error:   err.Error(),
		})
		return
	}

	// 重新获取更新后的番剧信息
	if err := models.DB.First(&bangumi, uint(bangumiID)).Error; err != nil {
		utils.LogError(fmt.Sprintf("获取更新后的番剧[%d]信息失败", bangumiID), err)
		c.JSON(http.StatusInternalServerError, BangumiResponse{
			Code:    http.StatusInternalServerError,
			Message: "获取更新后的番剧信息失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, BangumiResponse{
		Code:    http.StatusOK,
		Message: "更新番剧成功",
		Data:    bangumi,
	})
}

// GetUserFavorites godoc
// @Summary      获取用户收藏的番剧列表
// @Description  获取当前登录用户收藏的所有番剧
// @Tags         番剧
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        page query int false "页码，默认1"
// @Param        page_size query int false "每页数量，默认10"
// @Success      200  {object}  Response
// @Failure      401  {object}  Response
// @Failure      500  {object}  Response
// @Router       /user/favorites [get]
func GetUserFavorites(c *gin.Context) {
	userId, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{Error: "用户未认证"})
		return
	}

	// 获取分页参数
	page := utils.GetPage(c)
	pageSize := utils.GetPageSize(c)

	// 查询用户收藏的番剧
	var favorites []models.BangumiFavorite
	var total int64

	// 获取总数
	if err := models.DB.Model(&models.BangumiFavorite{}).Where("user_id = ?", userId).Count(&total).Error; err != nil {
		utils.LogError("获取用户收藏总数失败", err)
		c.JSON(http.StatusInternalServerError, Response{Error: "获取收藏列表失败"})
		return
	}

	// 获取收藏列表
	if err := models.DB.Where("user_id = ?", userId).
		Preload("Bangumi").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&favorites).Error; err != nil {
		utils.LogError("获取用户收藏列表失败", err)
		c.JSON(http.StatusInternalServerError, Response{Error: "获取收藏列表失败"})
		return
	}

	clientIP := c.ClientIP()
	fmt.Printf("GetUserFavorites - Client IP: %s\n", clientIP) // 添加日志

	// 构建响应数据
	bangumiList := make([]gin.H, 0) // list 为空的情况
	for _, fav := range favorites {
		bangumiList = append(bangumiList, gin.H{
			"id":             fav.Bangumi.ID,
			"title":          fav.Bangumi.OfficialTitle,
			"cover":          utils.GetPrefixedURL(clientIP, fav.Bangumi.PosterLink),
			"description":    "",
			"year":           fav.Bangumi.Year,
			"season":         fav.Bangumi.Season,
			"status":         "",
			"favorite_at":    fav.CreatedAt,
			"view_count":     fav.Bangumi.ViewCount,     // 添加播放量
			"favorite_count": fav.Bangumi.FavoriteCount, // 添加收藏量
		})
	}

	c.JSON(http.StatusOK, Response{
		Data: gin.H{
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
			"list":        bangumiList,
		},
	})
}
