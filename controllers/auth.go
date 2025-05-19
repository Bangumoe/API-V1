package controllers

import (
	"backend/models"
	"backend/services/activity"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type AuthController struct {
	DB              *gorm.DB
	activityService *activity.ActivityService
}

func NewAuthController(db *gorm.DB, activityService *activity.ActivityService) *AuthController {
	return &AuthController{
		DB:              db,
		activityService: activityService,
	}
}

// Response 通用响应结构
type Response struct {
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token   string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	Message string `json:"message" example:"登录成功"`
	Role    string `json:"role" example:"regular"`
}

// Register godoc
// @Summary      用户注册
// @Description  注册新用户
// @Tags         认证
// @Accept       multipart/form-data
// @Produce      json
// @Param        username formData string true "用户名"
// @Param        password formData string true "密码"
// @Param        email formData string true "邮箱"
// @Param        role formData string false "角色"
// @Param        avatar formData file false "头像文件"
// @Success      200  {object}  Response
// @Failure      400  {object}  Response
// @Router       /register [post]
func (ac *AuthController) Register(c *gin.Context) {
	// 获取表单数据
	username := c.PostForm("username")
	password := c.PostForm("password")
	email := c.PostForm("email")
	role := c.PostForm("role")

	// 验证必需字段
	if username == "" || password == "" || email == "" {
		c.JSON(http.StatusBadRequest, Response{Error: "用户名、密码和邮箱不能为空"})
		return
	}

	// 处理头像上传
	avatarPath := ""
	file, err := c.FormFile("avatar")
	if err == nil && file != nil {
		// 生成唯一文件名
		ext := filepath.Ext(file.Filename)
		fileName := fmt.Sprintf("avatar_%d%s", time.Now().UnixNano(), ext)
		avatarDir := "uploads/avatars"

		// 确保目录存在
		if err := os.MkdirAll(avatarDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, Response{Error: "创建头像目录失败"})
			return
		}

		// 保存文件
		filePath := filepath.Join(avatarDir, fileName)
		if err := c.SaveUploadedFile(file, filePath); err != nil {
			c.JSON(http.StatusInternalServerError, Response{Error: "保存头像失败"})
			return
		}
		avatarPath = "/" + filePath
	}

	// 设置默认角色
	userRole := models.RoleRegular
	if role != "" {
		switch role {
		case models.RoleAdmin, models.RolePremium, models.RoleRegular:
			userRole = role
		default:
			c.JSON(http.StatusBadRequest, Response{Error: "无效的用户角色"})
			return
		}
	}

	user := models.User{
		Username: username,
		Password: password,
		Email:    email,
		Role:     userRole,
		Avatar:   avatarPath,
	}

	if err := user.HashPassword(); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: "密码加密失败"})
		return
	}

	if err := ac.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "用户名或邮箱已存在"})
		return
	}

	// 记录注册活动
	ac.activityService.RecordActivity("user", fmt.Sprintf("新用户 \"%s\" 注册成功", username))

	c.JSON(http.StatusOK, Response{
		Message: "注册成功",
		Data: gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"role":       user.Role,
			"avatar":     user.Avatar,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		},
	})
}

// Login godoc
// @Summary      用户登录
// @Description  用户登录并获取token
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        login body LoginRequest true "登录信息"
// @Success      200  {object}  LoginResponse
// @Failure      400  {object}  Response
// @Failure      401  {object}  Response
// @Router       /login [post]
func (ac *AuthController) Login(c *gin.Context) {
	var loginUser LoginRequest
	if err := c.ShouldBindJSON(&loginUser); err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "请求数据格式不正确"})
		return
	}

	// 添加输入验证
	if loginUser.Username == "" || loginUser.Password == "" {
		c.JSON(http.StatusBadRequest, Response{Error: "用户名和密码不能为空"})
		return
	}

	fmt.Printf("登录请求 - 用户名: %s\n", loginUser.Username)

	var user models.User
	if err := ac.DB.Where("username = ?", loginUser.Username).First(&user).Error; err != nil {
		fmt.Printf("数据库查询错误: %v\n", err)
		c.JSON(http.StatusUnauthorized, Response{Error: "用户名或密码错误"})
		return
	}

	fmt.Printf("用户输入密码: %s\n", loginUser.Password)
	fmt.Printf("数据库存储的加密密码: %s\n", user.Password)

	if err := user.ComparePassword(loginUser.Password); err != nil {
		fmt.Printf("密码验证失败: %v\n", err)
		c.JSON(http.StatusUnauthorized, Response{Error: "用户名或密码错误"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role, // 在 JWT 中添加角色信息
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: "生成令牌失败"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token:   tokenString,
		Message: "登录成功",
		Role:    user.Role,
	})
}

// GetUserInfo godoc
// @Summary      获取当前用户信息
// @Description  使用token获取当前登录用户的详细信息
// @Tags         用户
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Success      200  {object}  Response
// @Failure      401  {object}  Response
// @Router       /user/info [get]
func (ac *AuthController) GetUserInfo(c *gin.Context) {
	userId, _ := c.Get("user_id")
	var user models.User
	if err := ac.DB.First(&user, userId).Error; err != nil {
		c.JSON(http.StatusUnauthorized, Response{Error: "获取用户信息失败"})
		return
	}

	c.JSON(http.StatusOK, Response{
		Data: gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"role":       user.Role,
			"avatar":     user.Avatar,
			"is_allowed": user.IsAllowed,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		},
	})
}

// LoginRequest represents login request body
type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"user123"`
	Password string `json:"password" binding:"required" example:"password123"`
}
