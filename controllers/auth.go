package controllers

import (
	"backend/models"
	"backend/services/activity"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"backend/utils"

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

	// 获取用户收藏番剧数量
	var favoriteCount int64
	if err := ac.DB.Model(&models.BangumiFavorite{}).Where("user_id = ?", userId).Count(&favoriteCount).Error; err != nil {
		utils.LogError("获取用户收藏数量失败", err)
	}

	// 获取用户评论数量
	var commentCount int64
	if err := ac.DB.Model(&models.BangumiRating{}).Where("user_id = ?", userId).Count(&commentCount).Error; err != nil {
		utils.LogError("获取用户评论数量失败", err)
	}

	c.JSON(http.StatusOK, Response{
		Data: gin.H{
			"id":             user.ID,
			"username":       user.Username,
			"email":          user.Email,
			"role":           user.Role,
			"avatar":         user.Avatar,
			"is_allowed":     user.IsAllowed,
			"created_at":     user.CreatedAt,
			"updated_at":     user.UpdatedAt,
			"favorite_count": favoriteCount,
			"comment_count":  commentCount,
		},
	})
}

// UpdateUserInfo godoc
// @Summary      更新当前用户信息
// @Description  更新当前登录用户的基本信息（邮箱、头像和密码）
// @Tags         用户
// @Accept       multipart/form-data
// @Produce      json
// @Security     Bearer
// @Param        email formData string false "邮箱"
// @Param        avatar formData file false "头像文件"
// @Param        old_password formData string false "旧密码"
// @Param        new_password formData string false "新密码"
// @Success      200  {object}  Response
// @Failure      400  {object}  Response
// @Failure      401  {object}  Response
// @Failure      500  {object}  Response
// @Router       /user/info [put]
func (ac *AuthController) UpdateUserInfo(c *gin.Context) {
	userId, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{Error: "用户未认证"})
		return
	}

	// 获取当前用户信息
	var user models.User
	if err := ac.DB.First(&user, userId).Error; err != nil {
		c.JSON(http.StatusUnauthorized, Response{Error: "获取用户信息失败"})
		return
	}

	// 获取表单数据
	email := c.PostForm("email")
	oldPassword := c.PostForm("old_password")
	newPassword := c.PostForm("new_password")

	// 验证邮箱是否已存在
	if email != "" && email != user.Email {
		var existingUser models.User
		if err := ac.DB.Where("email = ? AND id != ?", email, userId).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusBadRequest, Response{Error: "邮箱已存在"})
			return
		}
	}

	// 处理密码修改
	if oldPassword != "" && newPassword != "" {
		// 验证旧密码
		if err := user.ComparePassword(oldPassword); err != nil {
			c.JSON(http.StatusBadRequest, Response{Error: "旧密码错误"})
			return
		}

		// 验证新密码长度
		if len(newPassword) < 6 {
			c.JSON(http.StatusBadRequest, Response{Error: "新密码长度不能少于6个字符"})
			return
		}

		// 验证新密码不能与旧密码相同
		if oldPassword == newPassword {
			c.JSON(http.StatusBadRequest, Response{Error: "新密码不能与旧密码相同"})
			return
		}

		// 更新密码
		user.Password = newPassword
		if err := user.HashPassword(); err != nil {
			utils.LogError("密码加密失败", err)
			c.JSON(http.StatusInternalServerError, Response{Error: "密码更新失败"})
			return
		}
	}

	// 处理头像上传
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

		// 删除旧头像文件
		if user.Avatar != "" {
			oldAvatarPath := user.Avatar
			if oldAvatarPath[0] == '/' {
				oldAvatarPath = oldAvatarPath[1:]
			}
			if err := os.Remove(oldAvatarPath); err != nil {
				utils.LogError("删除旧头像文件失败", err)
			}
		}

		user.Avatar = "/" + filePath
	}

	// 更新用户信息
	updates := map[string]interface{}{}
	if email != "" {
		updates["email"] = email
	}
	if user.Avatar != "" {
		updates["avatar"] = user.Avatar
	}
	if user.Password != "" {
		updates["password"] = user.Password
	}

	if err := ac.DB.Model(&user).Updates(updates).Error; err != nil {
		utils.LogError("更新用户信息失败", err)
		c.JSON(http.StatusInternalServerError, Response{Error: "更新用户信息失败"})
		return
	}

	// 记录活动
	activityMsg := "用户更新了个人信息"
	if newPassword != "" {
		activityMsg = "用户更新了个人信息和密码"
	}
	ac.activityService.RecordActivity("user", fmt.Sprintf("用户 \"%s\" %s", user.Username, activityMsg))

	// 获取更新后的用户信息
	if err := ac.DB.First(&user, userId).Error; err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: "获取更新后的用户信息失败"})
		return
	}

	c.JSON(http.StatusOK, Response{
		Message: "更新用户信息成功",
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

// UpdatePasswordRequest 密码更新请求结构
type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required" example:"oldpass123"`
	NewPassword string `json:"new_password" binding:"required" example:"newpass123"`
}

// UpdatePassword godoc
// @Summary      修改密码
// @Description  修改当前登录用户的密码
// @Tags         用户
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        request body UpdatePasswordRequest true "密码更新信息"
// @Success      200  {object}  Response
// @Failure      400  {object}  Response
// @Failure      401  {object}  Response
// @Failure      500  {object}  Response
// @Router       /user/password [put]
func (ac *AuthController) UpdatePassword(c *gin.Context) {
	userId, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{Error: "用户未认证"})
		return
	}

	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "无效的请求参数"})
		return
	}

	// 验证新密码长度
	if len(req.NewPassword) < 6 {
		c.JSON(http.StatusBadRequest, Response{Error: "新密码长度不能少于6个字符"})
		return
	}

	// 获取当前用户信息
	var user models.User
	if err := ac.DB.First(&user, userId).Error; err != nil {
		c.JSON(http.StatusUnauthorized, Response{Error: "获取用户信息失败"})
		return
	}

	// 验证旧密码
	if err := user.ComparePassword(req.OldPassword); err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "旧密码错误"})
		return
	}

	// 验证新密码不能与旧密码相同
	if req.OldPassword == req.NewPassword {
		c.JSON(http.StatusBadRequest, Response{Error: "新密码不能与旧密码相同"})
		return
	}

	// 更新密码
	user.Password = req.NewPassword
	if err := user.HashPassword(); err != nil {
		utils.LogError("密码加密失败", err)
		c.JSON(http.StatusInternalServerError, Response{Error: "密码更新失败"})
		return
	}

	if err := ac.DB.Model(&user).Update("password", user.Password).Error; err != nil {
		utils.LogError("更新密码失败", err)
		c.JSON(http.StatusInternalServerError, Response{Error: "密码更新失败"})
		return
	}

	// 记录活动
	ac.activityService.RecordActivity("user", fmt.Sprintf("用户 \"%s\" 修改了密码", user.Username))

	c.JSON(http.StatusOK, Response{
		Message: "密码修改成功",
	})
}
