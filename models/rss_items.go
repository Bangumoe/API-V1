package models

import (
	"gorm.io/gorm"
)

type RSSItem struct {
	gorm.Model
	BangumiID   uint     `json:"bangumi_id" gorm:"index;not null" description:"关联番剧ID"`
	RssID       uint     `json:"rss_id" gorm:"index;not null" description:"关联RSS源ID"`
	Title       string   `json:"name" gorm:"type:varchar(255);not null" description:"番剧名"`
	URL         string   `json:"url" gorm:"type:varchar(511);not null;default:'https://example.com/torrent.torrent'" description:"种子URL"`
	Homepage    string   `json:"homepage,omitempty" gorm:"type:varchar(511)" description:"主页URL"`
	Downloaded  bool     `json:"downloaded" gorm:"default:false" description:"下载状态"`
	Episode     *float64 `json:"episode" description:"集数"`
	Resolution  string   `json:"resolution,omitempty" gorm:"type:varchar(50)" description:"分辨率"`
	Source      string   `json:"source,omitempty" gorm:"type:varchar(100)" description:"来源"`
	Group       string   `json:"group,omitempty" gorm:"type:varchar(100)" description:"字幕组"`
	ReleaseDate string   `json:"release_date,omitempty" gorm:"type:varchar(50)" description:"发布日期"`

	// 更新外键配置
	RssFeed RSSFeed `gorm:"foreignKey:RssID;references:ID"`
	Bangumi Bangumi `gorm:"foreignKey:BangumiID;references:ID"`
}

func (RSSItem) TableName() string {
	return "rss_items" // 确认表名
}

// BeforeCreate 钩子函数（可根据实际需求补充逻辑）
func (i *RSSItem) BeforeCreate(tx *gorm.DB) error {
	return nil
}
