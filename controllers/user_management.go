package controllers

import (
	"backend/models"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserManagementController struct {
	DB *gorm.DB
}

func NewUserManagementController(db *gorm.DB) *UserManagementController {
	return &UserManagementController{DB: db}
}

// GetAllUsers godoc
// @Summary      获取所有用户
// @Description  获取系统中所有用户的信息
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Success      200  {object}  Response
// @Failure      401  {object}  Response
// @Failure      403  {object}  Response
// @Router       /admin/users [get]
func (uc *UserManagementController) GetAllUsers(c *gin.Context) {
	var users []models.User
	if err := uc.DB.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: "获取用户列表失败"})
		return
	}
	c.JSON(http.StatusOK, Response{Data: users})
}

// GetUser godoc
// @Summary      获取单个用户
// @Description  通过用户ID获取特定用户信息
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "用户ID"
// @Security     Bearer
// @Success      200  {object}  Response
// @Failure      401  {object}  Response
// @Failure      403  {object}  Response
// @Failure      404  {object}  Response
// @Router       /admin/users/{id} [get]
func (uc *UserManagementController) GetUser(c *gin.Context) {
	id := c.Param("id")
	var user models.User
	if err := uc.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, Response{Error: "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, Response{Data: user})
}

// UpdateUser godoc
// @Summary      更新用户信息
// @Description  更新指定用户的信息
// @Tags         用户管理
// @Accept       multipart/form-data
// @Produce      json
// @Param        id   path      int  true  "用户ID"
// @Param        username formData string false "用户名"
// @Param        password formData string false "密码"
// @Param        email formData string false "邮箱"
// @Param        role formData string false "角色"
// @Param        is_allowed formData string false "是否允许访问"
// @Param        avatar formData file false "头像文件"
// @Security     Bearer
// @Success      200  {object}  Response
// @Failure      400  {object}  Response
// @Failure      401  {object}  Response
// @Failure      403  {object}  Response
// @Failure      404  {object}  Response
// @Router       /admin/users/{id} [put]
func (uc *UserManagementController) UpdateUser(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if err := uc.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, Response{Error: "用户不存在"})
		return
	}

	// 获取表单数据
	if username := c.PostForm("username"); username != "" {
		user.Username = username
	}
	if email := c.PostForm("email"); email != "" {
		user.Email = email
	}
	if password := c.PostForm("password"); password != "" {
		user.Password = password
		if err := user.HashPassword(); err != nil {
			c.JSON(http.StatusInternalServerError, Response{Error: "密码加密失败"})
			return
		}
	}
	if role := c.PostForm("role"); role != "" {
		switch role {
		case models.RoleAdmin, models.RolePremium, models.RoleRegular:
			user.Role = role
			// 如果是管理员，自动设置is_allowed为true
			if role == models.RoleAdmin {
				user.IsAllowed = true
			}
		default:
			c.JSON(http.StatusBadRequest, Response{Error: "无效的用户角色"})
			return
		}
	}

	// 处理is_allowed字段
	if isAllowed := c.PostForm("is_allowed"); isAllowed != "" {
		allowed, err := strconv.ParseBool(isAllowed)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{Error: "无效的is_allowed值"})
			return
		}
		// 如果是管理员，不允许关闭is_allowed
		if user.Role == models.RoleAdmin && !allowed {
			c.JSON(http.StatusBadRequest, Response{Error: "管理员不能关闭内测访问权限"})
			return
		}
		user.IsAllowed = allowed
	}

	// 处理头像上传
	file, err := c.FormFile("avatar")
	if err == nil && file != nil {
		// 删除旧头像
		if user.Avatar != "" {
			oldAvatarPath := filepath.Join(".", user.Avatar)
			if err := os.Remove(oldAvatarPath); err != nil {
				fmt.Printf("删除旧头像失败: %v\n", err)
			}
		}

		// 生成新头像文件名
		ext := filepath.Ext(file.Filename)
		fileName := fmt.Sprintf("avatar_%d%s", time.Now().UnixNano(), ext)
		avatarDir := "uploads/avatars"

		// 确保目录存在
		if err := os.MkdirAll(avatarDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, Response{Error: "创建头像目录失败"})
			return
		}

		// 保存新头像
		filePath := filepath.Join(avatarDir, fileName)
		if err := c.SaveUploadedFile(file, filePath); err != nil {
			c.JSON(http.StatusInternalServerError, Response{Error: "保存头像失败"})
			return
		}
		user.Avatar = "/" + filePath
	}

	if err := uc.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: "更新用户失败"})
		return
	}

	c.JSON(http.StatusOK, Response{Message: "用户更新成功", Data: user})
}

// DeleteUser godoc
// @Summary      删除用户
// @Description  删除指定的用户
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "用户ID"
// @Security     Bearer
// @Success      200  {object}  Response
// @Failure      401  {object}  Response
// @Failure      403  {object}  Response
// @Failure      404  {object}  Response
// @Router       /admin/users/{id} [delete]
func (uc *UserManagementController) DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "无效的用户ID"})
		return
	}

	// 避免删除自己
	currentUserId := c.MustGet("user_id").(float64)
	if int(currentUserId) == id {
		c.JSON(http.StatusBadRequest, Response{Error: "不能删除自己的账号"})
		return
	}

	// 使用 Unscoped().Delete 来永久删除记录
	result := uc.DB.Unscoped().Delete(&models.User{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: "删除用户失败"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, Response{Error: "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, Response{Message: "用户删除成功"})
}
