package rss

import (
	"backend/utils"
	"time"

	"gorm.io/gorm"
)

// RSSUpdateScheduler RSS更新调度器
type RSSUpdateScheduler struct {
	db           *gorm.DB
	isRunning    bool
	stopChan     chan bool
	completeChan chan bool
}

// NewRSSUpdateScheduler 创建新调度器
func NewRSSUpdateScheduler(db *gorm.DB) *RSSUpdateScheduler {
	return &RSSUpdateScheduler{
		db:           db,
		isRunning:    false,
		stopChan:     make(chan bool),
		completeChan: make(chan bool),
	}
}

// Start 启动RSS更新调度器
func (s *RSSUpdateScheduler) Start() {
	if s.isRunning {
		utils.LogInfo("RSS更新调度器已在运行中")
		return
	}

	s.isRunning = true
	utils.LogInfo("RSS更新调度器已启动，按订阅源配置的更新间隔运行")

	go func() {
		for {
			select {
			case <-time.After(time.Minute):
				s.updateRSS()
			case <-s.stopChan:
				utils.LogInfo("RSS更新调度器已停止")
				s.isRunning = false
				s.completeChan <- true
				return
			}
		}
	}()
}

// Stop 停止RSS更新调度器
func (s *RSSUpdateScheduler) Stop() {
	if !s.isRunning {
		return
	}

	s.stopChan <- true
	<-s.completeChan
}

// IsRunning 检查调度器是否正在运行
func (s *RSSUpdateScheduler) IsRunning() bool {
	return s.isRunning
}

// updateRSS 执行RSS更新
func (s *RSSUpdateScheduler) updateRSS() {
	utils.LogInfo("开始执行RSS更新任务")
	utils.LogInfo("准备调用 UpdateRSSFeeds 函数")
	err := UpdateRSSFeeds(s.db, false)
	if err != nil {
		utils.LogError("RSS更新任务执行失败", err)
		return
	}
	utils.LogInfo("UpdateRSSFeeds 函数调用成功")
	utils.LogInfo("RSS更新任务执行完成")
}
