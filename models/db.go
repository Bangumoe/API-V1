package models

import (
	"gorm.io/gorm"
)

// DB 全局数据库连接实例
var DB *gorm.DB

// SetDB 设置全局数据库连接
func SetDB(db *gorm.DB) {
	DB = db
}
