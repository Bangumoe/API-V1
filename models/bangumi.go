package models

import (
	"gorm.io/gorm"
)

type Bangumi struct {
	gorm.Model
	PosterHash    *string `gorm:"type:varchar(32);index:idx_poster_hash;uniqueIndex:uniq_poster_hash_season;comment:海报文件的MD5哈希值" json:"poster_hash"`
	OfficialTitle string  `gorm:"type:varchar(255);not null;comment:番剧中文名;uniqueIndex:uniq_poster_hash_season" json:"official_title"`
	Year          *string `gorm:"type:varchar(4);comment:番剧年份" json:"year,omitempty"`
	Season        int     `gorm:"default:1;comment:番剧季度;uniqueIndex:uniq_poster_hash_season" json:"season"`
	Source        *string `gorm:"type:varchar(100);comment:来源" json:"source,omitempty"`
	PosterLink    *string `gorm:"type:varchar(255);comment:海报链接" json:"poster_link,omitempty"`
}

// BangumiCreateRequest 创建用请求结构体
type BangumiCreateRequest struct {
	OfficialTitle string  `json:"official_title" binding:"required"`
	Year          *string `json:"year"`
	Season        int     `json:"season"`
	PosterLink    *string `json:"poster_link"`
}

// BangumiUpdateRequest 更新用请求结构体
type BangumiUpdateRequest struct {
	OfficialTitle string  `json:"official_title"`
	Year          *string `json:"year"`
	Season        int     `json:"season"`
	PosterLink    *string `json:"poster_link"`
}

// BangumiResponse 响应结构体
type BangumiResponse struct {
	ID            uint    `json:"id"`
	OfficialTitle string  `json:"official_title"`
	TitleRaw      string  `json:"title_raw"`
	Season        int     `json:"season"`
	PosterLink    *string `json:"poster_link,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

func (Bangumi) TableName() string {
	return "bangumi"
}
