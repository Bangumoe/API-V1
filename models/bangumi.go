package models

import (
	"gorm.io/gorm"
)

type Bangumi struct {
	gorm.Model
	PosterHash    *string `gorm:"type:varchar(32);index:idx_poster_hash;uniqueIndex:uniq_poster_hash_season;comment:海报文件的MD5哈希值 (此字段已弃用，将设置为NULL)" json:"poster_hash,omitempty"`
	OfficialTitle string  `gorm:"type:varchar(255);not null;comment:番剧中文名;uniqueIndex:uniq_poster_hash_season" json:"official_title"`
	Year          *string `gorm:"type:varchar(4);comment:番剧年份" json:"year,omitempty"`
	Season        int     `gorm:"default:1;comment:番剧季度;uniqueIndex:uniq_poster_hash_season" json:"season"`
	Source        *string `gorm:"type:varchar(100);comment:来源" json:"source,omitempty"`
	PosterLink    *string `gorm:"type:varchar(255);comment:海报链接" json:"poster_link,omitempty"`
	ViewCount     int64   `gorm:"default:0;comment:点击量" json:"view_count"`
	FavoriteCount int64   `gorm:"default:0;comment:收藏量" json:"favorite_count"`
	RatingAvg     float64 `gorm:"type:decimal(4,2);default:0;comment:平均评分" json:"rating_avg"`
	RatingCount   int64   `gorm:"default:0;comment:评分人数" json:"rating_count"`
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
	ViewCount     int64   `json:"view_count"`
	FavoriteCount int64   `json:"favorite_count"`
	RatingAvg     float64 `json:"rating_avg"`
	RatingCount   int64   `json:"rating_count"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

func (Bangumi) TableName() string {
	return "bangumi"
}
