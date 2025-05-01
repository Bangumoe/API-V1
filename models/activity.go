package models

import (
	"time"

	"gorm.io/gorm"
)

// Activity 活动记录模型
// @Description 系统活动记录
type Activity struct {
	ID        uint           `json:"id" gorm:"primarykey" example:"1"`
	Type      string         `json:"type" gorm:"type:varchar(50);not null" example:"user" description:"活动类型(user/rss/bangumi/system)"`
	Content   string         `json:"content" gorm:"type:text;not null" example:"新用户 \"AnimeUser\" 注册成功" description:"活动内容"`
	CreatedAt time.Time      `json:"created_at" example:"2024-01-20T15:04:05Z" description:"创建时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Activity) TableName() string {
	return "activities"
}
