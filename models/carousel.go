package models

import (
	"time"
)

// CarouselResponse 轮播图响应结构体
// @Description 轮播图响应数据
type CarouselResponse struct {
	// @Description 轮播图ID
	ID uint `json:"id"`
	// @Description 创建时间
	CreatedAt time.Time `json:"created_at"`
	// @Description 更新时间
	UpdatedAt time.Time `json:"updated_at"`
	// @Description 轮播图标题
	Title string `json:"title"`
	// @Description 轮播图副标题
	Subtitle string `json:"subtitle"`
	// @Description 轮播图详细描述
	Description string `json:"description"`
	// @Description 轮播图图片URL
	ImageURL string `json:"image_url"`
	// @Description 轮播图点击跳转链接
	Link string `json:"link"`
	// @Description 轮播图显示顺序
	Order int `json:"order"`
	// @Description 轮播图是否激活
	IsActive bool `json:"is_active"`
	// @Description 轮播图开始显示时间
	StartDate time.Time `json:"start_date"`
	// @Description 轮播图结束显示时间
	EndDate time.Time `json:"end_date"`
}

// Carousel 轮播图模型
// @Description 网站首页轮播图数据模型
type Carousel struct {
	ID          uint       `json:"-" gorm:"primarykey"`
	CreatedAt   time.Time  `json:"-"`
	UpdatedAt   time.Time  `json:"-"`
	Title       string     `json:"-" gorm:"type:varchar(255);not null"`
	Subtitle    string     `json:"-" gorm:"type:varchar(255)"`
	Description string     `json:"-" gorm:"type:text"`
	ImageURL    string     `json:"-" gorm:"type:varchar(255);not null"`
	Link        string     `json:"-" gorm:"type:varchar(255)"`
	Order       int        `json:"-" gorm:"type:int;default:0"`
	IsActive    bool       `json:"-" gorm:"default:true"`
	StartDate   *time.Time `json:"-"`
	EndDate     *time.Time `json:"-"`
}

// ToResponse 将模型转换为响应结构体
func (c *Carousel) ToResponse() CarouselResponse {
	var startDate, endDate time.Time
	if c.StartDate != nil {
		startDate = *c.StartDate
	}
	if c.EndDate != nil {
		endDate = *c.EndDate
	}

	return CarouselResponse{
		ID:          c.ID,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		Title:       c.Title,
		Subtitle:    c.Subtitle,
		Description: c.Description,
		ImageURL:    c.ImageURL,
		Link:        c.Link,
		Order:       c.Order,
		IsActive:    c.IsActive,
		StartDate:   startDate,
		EndDate:     endDate,
	}
}
