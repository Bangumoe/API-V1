package models

import (
	"gorm.io/gorm"
)

// BangumiFavorite 番剧收藏模型
type BangumiFavorite struct {
	gorm.Model
	UserID    uint    `gorm:"index:idx_user_bangumi;uniqueIndex:uniq_user_bangumi" json:"user_id"`
	BangumiID uint    `gorm:"index:idx_user_bangumi;uniqueIndex:uniq_user_bangumi" json:"bangumi_id"`
	User      User    `gorm:"foreignKey:UserID" json:"-"`
	Bangumi   Bangumi `gorm:"foreignKey:BangumiID" json:"-"`
}

// BangumiRating 番剧评分模型
type BangumiRating struct {
	gorm.Model
	UserID    uint    `gorm:"index:idx_user_bangumi_rating;uniqueIndex:uniq_user_bangumi_rating" json:"user_id"`
	BangumiID uint    `gorm:"index:idx_user_bangumi_rating;uniqueIndex:uniq_user_bangumi_rating" json:"bangumi_id"`
	Score     float64 `gorm:"type:decimal(3,1);not null;check:score >= 0 AND score <= 10" json:"score"` // 评分范围0-10
	Comment   string  `gorm:"type:text" json:"comment"`                                                 // 评价内容
	User      User    `gorm:"foreignKey:UserID" json:"-"`
	Bangumi   Bangumi `gorm:"foreignKey:BangumiID" json:"-"`
}

// BangumiRatingRequest 评分请求结构体
type BangumiRatingRequest struct {
	Score   float64 `json:"score" binding:"required,min=0,max=10"`
	Comment string  `json:"comment"`
}

// BangumiRatingResponse 评分响应结构体
type BangumiRatingResponse struct {
	ID        uint    `json:"id"`
	UserID    uint    `json:"user_id"`
	BangumiID uint    `json:"bangumi_id"`
	Score     float64 `json:"score"`
	Comment   string  `json:"comment,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// TableName 设置BangumiFavorite表名
func (BangumiFavorite) TableName() string {
	return "bangumi_favorites"
}

// TableName 设置BangumiRating表名
func (BangumiRating) TableName() string {
	return "bangumi_ratings"
}
