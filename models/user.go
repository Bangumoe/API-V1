package models

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 用户角色常量
const (
	RoleAdmin   = "admin"   // 管理员
	RolePremium = "premium" // 高级会员
	RoleRegular = "regular" // 普通会员
)

// UserRequest 用于 Swagger 文档的用户请求模型
type UserRequest struct {
	Username string `json:"username" example:"user123" binding:"required" description:"用户名"`
	Password string `json:"password" example:"password123" binding:"required" description:"用户密码"`
	Email    string `json:"email" example:"user@example.com" binding:"required,email" description:"邮箱地址"`
	Role     string `json:"role,omitempty" example:"regular" description:"用户角色"`
	Avatar   string `json:"avatar" example:"/uploads/avatars/default.jpg" description:"头像路径"`
}

// UserUpdateRequest 用于更新用户信息的请求模型
type UserUpdateRequest struct {
	Username string `json:"username" example:"user123" description:"用户名"`
	Password string `json:"password" example:"password123" description:"用户密码"`
	Email    string `json:"email" example:"user@example.com" description:"邮箱地址"`
	Role     string `json:"role,omitempty" example:"regular" description:"用户角色"`
	Avatar   string `json:"avatar" example:"/uploads/avatars/default.jpg" description:"头像路径"`
}

// User 用户模型（数据库模型）
type User struct {
	gorm.Model        // 这会自动包含 ID、CreatedAt、UpdatedAt、DeletedAt
	Username   string `json:"username" gorm:"type:varchar(50);uniqueIndex;not null"`
	Password   string `json:"-" gorm:"type:varchar(255);not null"`
	Email      string `json:"email" gorm:"type:varchar(100);uniqueIndex;not null"`
	Role       string `json:"role" gorm:"type:varchar(20);default:'regular'"` // 添加角色字段
	Avatar     string `json:"avatar" gorm:"type:varchar(255)"`                // 添加头像字段
}

func (u *User) HashPassword() error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

func (u *User) ComparePassword(password string) error {
	fmt.Printf("正在比较密码 - 输入: %s, 存储: %s\n", password, u.Password)
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
}
