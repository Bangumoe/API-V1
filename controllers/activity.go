package controllers

import (
	"backend/services/activity"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ActivityController struct {
	activityService *activity.ActivityService
}

func NewActivityController(activityService *activity.ActivityService) *ActivityController {
	return &ActivityController{
		activityService: activityService,
	}
}

// GetRecentActivities godoc
// @Summary      获取最近活动记录
// @Description  获取系统中最近的活动记录，包括用户操作、RSS更新等
// @Tags         系统管理
// @Accept       json
// @Produce      json
// @Param        limit  query    int     false  "返回记录数量限制(默认20)"  minimum(1) maximum(100)
// @Success      200    {object} Response{data=[]models.Activity}
// @Failure      401    {object} Response "未授权"
// @Failure      500    {object} Response "服务器错误"
// @Security     Bearer
// @Router       /activities [get]
func (ac *ActivityController) GetRecentActivities(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	activities, err := ac.activityService.GetRecentActivities(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Error: "获取活动记录失败",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Data: activities,
	})
}
