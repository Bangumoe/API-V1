package controllers

import (
	"backend/models"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateCarouselRequest 创建轮播图请求参数
// @Description 创建轮播图的请求参数
type CreateCarouselRequest struct {
	// @Description 轮播图标题
	Title string `form:"title" binding:"required"`
	// @Description 轮播图副标题
	Subtitle string `form:"subtitle"`
	// @Description 轮播图详细描述
	Description string `form:"description"`
	// @Description 轮播图图片文件
	ImageFile *multipart.FileHeader `form:"image_file" binding:"required"`
	// @Description 轮播图点击跳转链接
	Link string `form:"link"`
	// @Description 轮播图显示顺序
	Order int `form:"order"`
	// @Description 轮播图是否激活
	IsActive bool `form:"is_active"`
	// @Description 轮播图开始显示时间 (格式: 2006-01-02)
	StartDate string `form:"start_date"`
	// @Description 轮播图结束显示时间 (格式: 2006-01-02)
	EndDate string `form:"end_date"`
}

// UpdateCarouselRequest 更新轮播图请求参数
// @Description 更新轮播图的请求参数
type UpdateCarouselRequest struct {
	// @Description 轮播图标题
	Title *string `form:"title"`
	// @Description 轮播图副标题
	Subtitle *string `form:"subtitle"`
	// @Description 轮播图详细描述
	Description *string `form:"description"`
	// @Description 轮播图图片文件
	ImageFile *multipart.FileHeader `form:"image_file"`
	// @Description 轮播图点击跳转链接
	Link *string `form:"link"`
	// @Description 轮播图显示顺序
	Order *int `form:"order"`
	// @Description 轮播图是否激活
	IsActive *bool `form:"is_active"`
	// @Description 轮播图开始显示时间 (格式: 2006-01-02)
	StartDate *string `form:"start_date"`
	// @Description 轮播图结束显示时间 (格式: 2006-01-02)
	EndDate *string `form:"end_date"`
}

// CarouselOrderRequest 更新轮播图顺序请求参数
// @Description 更新轮播图顺序的请求参数
type CarouselOrderRequest struct {
	// @Description 轮播图ID
	ID uint `json:"id" binding:"required"`
	// @Description 轮播图顺序
	Order int `json:"order" binding:"required"`
}

// ErrorResponse 错误响应结构体
// @Description 错误响应
type ErrorResponse struct {
	// @Description 错误信息
	Error string `json:"error"`
}

// SuccessResponse 成功响应结构体
// @Description 成功响应
type SuccessResponse struct {
	// @Description 成功信息
	Message string `json:"message"`
}

// CarouselOrder 轮播图顺序结构体
// @Description 轮播图顺序信息
type CarouselOrder struct {
	// @Description 轮播图ID
	ID uint `json:"id"`
	// @Description 轮播图顺序
	Order int `json:"order"`
}

type CarouselController struct {
	DB *gorm.DB
}

func NewCarouselController(db *gorm.DB) *CarouselController {
	return &CarouselController{DB: db}
}

// CreateCarousel 创建新的轮播图
// @Summary 创建新的轮播图
// @Description 创建新的轮播图记录
// @Tags carousel
// @Accept multipart/form-data
// @Produce json
// @Param title formData string true "轮播图标题"
// @Param subtitle formData string false "轮播图副标题"
// @Param description formData string false "轮播图详细描述"
// @Param image_file formData file true "轮播图图片文件"
// @Param link formData string false "轮播图点击跳转链接"
// @Param order formData integer false "轮播图显示顺序"
// @Param is_active formData boolean false "轮播图是否激活"
// @Param start_date formData string false "轮播图开始显示时间"
// @Param end_date formData string false "轮播图结束显示时间"
// @Security Bearer
// @Success 201 {object} models.CarouselResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/carousels [post]
func (cc *CarouselController) CreateCarousel(c *gin.Context) {
	var req CreateCarouselRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// 处理文件上传
	file, err := req.ImageFile.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无法打开上传的文件"})
		return
	}
	defer file.Close()

	// 生成唯一的文件名
	ext := filepath.Ext(req.ImageFile.Filename)
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	uploadPath := filepath.Join("uploads", "carousels", filename)
	uploadDir := filepath.Join("uploads", "carousels")

	// 确保目录存在
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "无法创建上传目录"})
		return
	}

	// 保存文件
	if err := c.SaveUploadedFile(req.ImageFile, uploadPath); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "无法保存上传的文件"})
		return
	}

	// 构建图片URL
	imageURL := fmt.Sprintf("/uploads/carousels/%s", filename)

	carousel := models.Carousel{
		Title:       req.Title,
		Subtitle:    req.Subtitle,
		Description: req.Description,
		ImageURL:    imageURL,
		Link:        req.Link,
		Order:       req.Order,
		IsActive:    req.IsActive,
	}

	// 处理开始时间
	if req.StartDate != "" {
		startDate, err := time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "开始时间格式错误，应为 YYYY-MM-DD"})
			return
		}
		carousel.StartDate = &startDate
	}

	// 处理结束时间
	if req.EndDate != "" {
		endDate, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "结束时间格式错误，应为 YYYY-MM-DD"})
			return
		}
		carousel.EndDate = &endDate
	}

	if err := cc.DB.Create(&carousel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "创建轮播图失败"})
		return
	}

	c.JSON(http.StatusCreated, carousel.ToResponse())
}

// GetCarousels 获取所有轮播图
// @Summary 获取所有轮播图
// @Description 获取所有轮播图列表
// @Tags carousel
// @Produce json
// @Success 200 {array} models.CarouselResponse
// @Failure 500 {object} ErrorResponse
// @Router /carousels [get]
func (cc *CarouselController) GetCarousels(c *gin.Context) {
	var carousels []models.Carousel
	if err := cc.DB.Order("`order` asc").Find(&carousels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "获取轮播图列表失败"})
		return
	}

	response := make([]models.CarouselResponse, len(carousels))
	for i, carousel := range carousels {
		response[i] = carousel.ToResponse()
	}

	c.JSON(http.StatusOK, response)
}

// GetCarousel 获取单个轮播图
// @Summary 获取单个轮播图
// @Description 根据ID获取单个轮播图详情
// @Tags carousel
// @Produce json
// @Param id path int true "轮播图ID"
// @Security Bearer
// @Success 200 {object} models.CarouselResponse
// @Failure 404 {object} ErrorResponse
// @Router /admin/carousels/{id} [get]
func (cc *CarouselController) GetCarousel(c *gin.Context) {
	id := c.Param("id")
	var carousel models.Carousel

	if err := cc.DB.First(&carousel, id).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "轮播图不存在"})
		return
	}

	c.JSON(http.StatusOK, carousel.ToResponse())
}

// UpdateCarousel 更新轮播图
// @Summary 更新轮播图
// @Description 更新指定ID的轮播图信息
// @Tags carousel
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "轮播图ID"
// @Param title formData string false "轮播图标题"
// @Param subtitle formData string false "轮播图副标题"
// @Param description formData string false "轮播图详细描述"
// @Param image_file formData file false "轮播图图片文件"
// @Param link formData string false "轮播图点击跳转链接"
// @Param order formData integer false "轮播图显示顺序"
// @Param is_active formData boolean false "轮播图是否激活"
// @Param start_date formData string false "轮播图开始显示时间"
// @Param end_date formData string false "轮播图结束显示时间"
// @Security Bearer
// @Success 200 {object} models.CarouselResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/carousels/{id} [put]
func (cc *CarouselController) UpdateCarousel(c *gin.Context) {
	id := c.Param("id")
	var carousel models.Carousel

	if err := cc.DB.First(&carousel, id).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "轮播图不存在"})
		return
	}

	var req UpdateCarouselRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// 处理文件上传
	if req.ImageFile != nil {
		// 处理文件上传
		file, err := req.ImageFile.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无法打开上传的文件"})
			return
		}
		defer file.Close()

		// 生成唯一的文件名
		ext := filepath.Ext(req.ImageFile.Filename)
		filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
		uploadPath := filepath.Join("uploads", "carousels", filename)
		uploadDir := filepath.Join("uploads", "carousels")

		// 确保目录存在
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "无法创建上传目录"})
			return
		}

		// 保存文件
		if err := c.SaveUploadedFile(req.ImageFile, uploadPath); err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "无法保存上传的文件"})
			return
		}

		// 构建图片URL
		imageURL := fmt.Sprintf("/uploads/carousels/%s", filename)
		carousel.ImageURL = imageURL
	}

	// 只更新非空字段
	if req.Title != nil {
		carousel.Title = *req.Title
	}
	if req.Subtitle != nil {
		carousel.Subtitle = *req.Subtitle
	}
	if req.Description != nil {
		carousel.Description = *req.Description
	}
	if req.Link != nil {
		carousel.Link = *req.Link
	}
	if req.Order != nil {
		carousel.Order = *req.Order
	}
	if req.IsActive != nil {
		carousel.IsActive = *req.IsActive
	}

	// 处理开始时间
	if req.StartDate != nil {
		if *req.StartDate == "" {
			// 如果提供了空字符串，则设置为 NULL
			carousel.StartDate = nil
		} else {
			startDate, err := time.Parse("2006-01-02", *req.StartDate)
			if err != nil {
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: "开始时间格式错误，应为 YYYY-MM-DD"})
				return
			}
			carousel.StartDate = &startDate
		}
	}

	// 处理结束时间
	if req.EndDate != nil {
		if *req.EndDate == "" {
			// 如果提供了空字符串，则设置为 NULL
			carousel.EndDate = nil
		} else {
			endDate, err := time.Parse("2006-01-02", *req.EndDate)
			if err != nil {
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: "结束时间格式错误，应为 YYYY-MM-DD"})
				return
			}
			carousel.EndDate = &endDate
		}
	}

	if err := cc.DB.Save(&carousel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "更新轮播图失败"})
		return
	}

	c.JSON(http.StatusOK, carousel.ToResponse())
}

// DeleteCarousel 删除轮播图
// @Summary 删除轮播图
// @Description 删除指定ID的轮播图
// @Tags carousel
// @Produce json
// @Param id path int true "轮播图ID"
// @Security Bearer
// @Success 200 {object} SuccessResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/carousels/{id} [delete]
func (cc *CarouselController) DeleteCarousel(c *gin.Context) {
	id := c.Param("id")
	var carousel models.Carousel

	if err := cc.DB.First(&carousel, id).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "轮播图不存在"})
		return
	}

	// 使用Unscoped().Delete进行硬删除
	if err := cc.DB.Unscoped().Delete(&carousel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "删除轮播图失败"})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "轮播图已删除"})
}

// UpdateCarouselOrder 更新轮播图顺序
// @Summary 更新轮播图顺序
// @Description 批量更新轮播图的显示顺序
// @Tags carousel
// @Accept json
// @Produce json
// @Param orders body []CarouselOrderRequest true "轮播图顺序数组"
// @Security Bearer
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/carousels/order [put]
func (cc *CarouselController) UpdateCarouselOrder(c *gin.Context) {
	var orders []CarouselOrderRequest

	if err := c.ShouldBindJSON(&orders); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	for _, order := range orders {
		if err := cc.DB.Model(&models.Carousel{}).Where("id = ?", order.ID).Update("order", order.Order).Error; err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "更新顺序失败"})
			return
		}
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "顺序更新成功"})
}
