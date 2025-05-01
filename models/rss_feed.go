package models

import (
	"gorm.io/gorm"
)

// RSSFeedRequest 用于 Swagger 文档的RSS订阅源请求模型
type RSSFeedRequest struct {
	Name           string `json:"name" example:"莉可丽丝" binding:"required" description:"RSS源名称"`
	URL            string `json:"url" example:"https://mikanani.me/RSS/Bangumi?bangumiId=3644" binding:"required" description:"RSS源URL"`
	UpdateInterval int    `json:"update_interval" example:"1" binding:"required" description:"更新间隔（小时）"`
	Keywords       string `json:"keywords" example:"莉可丽丝,友谊是时间的窃贼" description:"关键词，多个关键词用逗号分隔"`
	Priority       int    `json:"priority" example:"0" description:"优先级"`
	ParserType     string `json:"parser_type" example:"mikanani" binding:"required" description:"解析器类型（mikanani/generic_rss）"`
}

// RSSFeedResponse 用于 Swagger 文档的RSS订阅源响应模型
type RSSFeedResponse struct {
	ID             uint   `json:"id" description:"RSS源ID"`
	Name           string `json:"name" description:"RSS源名称"`
	URL            string `json:"url" description:"RSS源URL"`
	UpdateInterval int    `json:"update_interval" description:"更新间隔（小时）"`
	Keywords       string `json:"keywords" description:"关键词，多个关键词用逗号分隔"`
	Priority       int    `json:"priority" description:"优先级"`
	ParserType     string `json:"parser_type" description:"解析器类型（mikanani/generic_rss）"`
	CreatedAt      string `json:"created_at" description:"创建时间"`
	UpdatedAt      string `json:"updated_at" description:"更新时间"`
}

// RSSFeed RSS订阅源模型（数据库模型）
type RSSFeed struct {
	gorm.Model            // 这会自动包含 ID、CreatedAt、UpdatedAt、DeletedAt
	Name           string `json:"name" gorm:"type:varchar(100);not null" description:"RSS源名称"`
	URL            string `json:"url" gorm:"type:varchar(255);not null;uniqueIndex" description:"RSS源URL"`
	UpdateInterval int    `json:"update_interval" gorm:"type:int;not null;default:1" description:"更新间隔（小时）"`
	Keywords       string `json:"keywords" gorm:"type:text" description:"关键词，多个关键词用逗号分隔"`
	Priority       int    `json:"priority" gorm:"type:int;default:0" description:"优先级"`
	ParserType     string `json:"parser_type" gorm:"type:varchar(20);not null;default:'mikanani'" description:"解析器类型（mikanani/generic_rss）"`
	// 可以添加与RSS条目的关联关系
	// Items []RSSItem `json:"items,omitempty" gorm:"foreignKey:FeedID"`
}

// TableName 指定表名
func (RSSFeed) TableName() string {
	return "rss_feeds"
}

// BeforeCreate 在创建记录前的钩子函数
func (f *RSSFeed) BeforeCreate(tx *gorm.DB) error {
	// 可以在这里添加一些创建前的验证或处理逻辑
	return nil
}

// BeforeUpdate 在更新记录前的钩子函数
func (f *RSSFeed) BeforeUpdate(tx *gorm.DB) error {
	// 可以在这里添加一些更新前的验证或处理逻辑
	return nil
}
