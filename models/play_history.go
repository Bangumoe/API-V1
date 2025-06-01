package models

import (
	"time"

	"gorm.io/gorm"
)

type PlayHistory struct {
	gorm.Model
	UpdatedAt  time.Time `gorm:"index"`
	UserId     uint      `gorm:"index:play_history_user_id_rss_items_id_IDX,unique" json:"user_id"`
	RssItemsId uint      `gorm:"index:play_history_user_id_rss_items_id_IDX,unique" json:"rss_items_id"`
}

func (PlayHistory) TableName() string {
	return "play_history" // 确认表名
}

// BeforeCreate 钩子函数（可根据实际需求补充逻辑）
func (i *PlayHistory) BeforeCreate(tx *gorm.DB) error {
	return nil
}
