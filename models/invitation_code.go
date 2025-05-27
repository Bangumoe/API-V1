package models

import (
	"time"

	"gorm.io/gorm"
)

// InvitationCode 存储邀请码信息
type InvitationCode struct {
	gorm.Model
	Code         string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"code"`    // 邀请码
	IsUsed       bool       `gorm:"default:false" json:"is_used"`                          // 是否已使用
	UsedByUserID *uint      `json:"used_by_user_id"`                                       // 使用者用户ID
	UsedByUser   *User      `gorm:"foreignKey:UsedByUserID" json:"used_by_user,omitempty"` // 关联用户 (可选)
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`                                  // 过期时间 (可选)
	GeneratedBy  *uint      `json:"generated_by"`                                          // 生成者ID (管理员)
	Generator    *User      `gorm:"foreignKey:GeneratedBy" json:"generator,omitempty"`     // 关联管理员 (可选)
}

// TableName 设置表名
func (InvitationCode) TableName() string {
	return "invitation_codes"
}
