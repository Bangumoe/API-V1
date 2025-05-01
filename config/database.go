package config

import (
	"backend/models"
	"fmt"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB() (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %v", err)
	}

	// 只进行表结构迁移，不删除现有数据
	err = db.AutoMigrate(
		&models.User{},
		&models.RSSFeed{},
		&models.Bangumi{},
		&models.RSSItem{},
		&models.Activity{},
	)
	if err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %v", err)
	}

	fmt.Println("数据库连接成功！")
	return db, nil
}
