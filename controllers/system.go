package controllers

import (
	"backend/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/host"
)

type SystemStatsResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type SystemStats struct {
	TotalBangumi  int64 `json:"total_bangumi"`
	TotalUsers    int64 `json:"total_users"`
	TotalRSSFeeds int64 `json:"total_rss_feeds"`
}

type SystemStatus struct {
	CPUUsage      float64        `json:"cpuUsage"`
	MemoryTotal   uint64         `json:"memoryTotal"`
	MemoryUsed    uint64         `json:"memoryUsed"`
	MemoryUsage   float64        `json:"memoryUsage"`
	DiskTotal     uint64         `json:"diskTotal"`
	DiskUsed      uint64         `json:"diskUsed"`
	DiskUsage     float64        `json:"diskUsage"`
	NetworkStatus NetworkMetrics `json:"networkStatus"`
	Uptime        float64        `json:"uptime"` // 添加系统运行时间字段
}

type NetworkMetrics struct {
	RxBytes     uint64 `json:"rxBytes"`
	TxBytes     uint64 `json:"txBytes"`
	Connections int    `json:"connections"`
}

// GetSystemStats 获取系统统计信息
// @Summary 获取系统统计信息
// @Description 获取系统中的番剧、用户和RSS订阅源总数统计
// @Tags 系统管理
// @Produce json
// @Success 200 {object} SystemStatsResponse
// @Router /admin/stats [get]
func GetSystemStats(c *gin.Context) {
	var stats SystemStats

	// 统计番剧总数
	models.DB.Model(&models.Bangumi{}).Count(&stats.TotalBangumi)

	// 统计用户总数
	models.DB.Model(&models.User{}).Count(&stats.TotalUsers)

	// 统计RSS订阅源总数
	models.DB.Model(&models.RSSFeed{}).Count(&stats.TotalRSSFeeds)

	c.JSON(http.StatusOK, SystemStatsResponse{
		Code:    http.StatusOK,
		Message: "获取系统统计信息成功",
		Data:    stats,
	})
}

// GetSystemStatus 获取系统状态信息
// @Summary 获取系统状态信息
// @Description 获取系统CPU、内存、磁盘和网络等实时状态信息
// @Tags 系统管理
// @Produce json
// @Success 200 {object} SystemStatsResponse
// @Router /admin/system/status [get]
func GetSystemStatus(c *gin.Context) {
	status := SystemStatus{}

	// 获取系统运行时间
	if uptime, err := host.Uptime(); err == nil {
		status.Uptime = float64(uptime)
	}

	// 获取CPU使用率
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		status.CPUUsage = cpuPercent[0]
	}

	// 获取内存信息
	if memInfo, err := mem.VirtualMemory(); err == nil {
		status.MemoryTotal = memInfo.Total
		status.MemoryUsed = memInfo.Used
		status.MemoryUsage = memInfo.UsedPercent
	}

	// 获取磁盘信息
	if diskInfo, err := disk.Usage("/"); err == nil {
		status.DiskTotal = diskInfo.Total
		status.DiskUsed = diskInfo.Used
		status.DiskUsage = diskInfo.UsedPercent
	}

	// 获取网络信息
	networkMetrics := NetworkMetrics{}
	if netStats, err := net.IOCounters(false); err == nil && len(netStats) > 0 {
		networkMetrics.RxBytes = netStats[0].BytesRecv
		networkMetrics.TxBytes = netStats[0].BytesSent
	}

	if connections, err := net.Connections("all"); err == nil {
		networkMetrics.Connections = len(connections)
	}

	status.NetworkStatus = networkMetrics

	c.JSON(http.StatusOK, SystemStatsResponse{
		Code:    http.StatusOK,
		Message: "获取系统状态信息成功",
		Data:    status,
	})
}
