package models

import (
	"gorm.io/gorm"
)

type PlayHistory struct {
	gorm.Model
	RssItemsId uint `json:"rss_items_id"`
	UserId     uint `json:"user_id"`
}

func (PlayHistory) TableName() string {
	return "play_history" // 确认表名
}

// BeforeCreate 钩子函数（可根据实际需求补充逻辑）
func (i *PlayHistory) BeforeCreate(tx *gorm.DB) error {
	return nil
}
