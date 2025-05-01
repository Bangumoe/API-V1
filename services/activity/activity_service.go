package activity

import (
	"backend/models"
	"backend/utils"

	"gorm.io/gorm"
)

// ActivityService 活动记录服务
type ActivityService struct {
	db *gorm.DB
}

func NewActivityService(db *gorm.DB) *ActivityService {
	return &ActivityService{db: db}
}

// RecordActivity 记录新的活动
func (s *ActivityService) RecordActivity(activityType string, content string) error {
	activity := models.Activity{
		Type:    activityType,
		Content: content,
	}

	if err := s.db.Create(&activity).Error; err != nil {
		utils.LogError("记录活动失败", err)
		return err
	}

	return nil
}

// GetRecentActivities 获取最近的活动记录
func (s *ActivityService) GetRecentActivities(limit int) ([]models.Activity, error) {
	var activities []models.Activity

	if err := s.db.Order("created_at DESC").Limit(limit).Find(&activities).Error; err != nil {
		utils.LogError("获取最近活动记录失败", err)
		return nil, err
	}

	return activities, nil
}
