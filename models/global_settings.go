package models

import (
	"gorm.io/gorm"
)

// GlobalSettings 存储全局设置
type GlobalSettings struct {
	gorm.Model
	GlobalKeywords    string `json:"global_keywords" gorm:"type:text" description:"全局关键词"`
	ExcludeKeywords   string `json:"exclude_keywords" gorm:"type:text" description:"全局排除关键词"`
	SubGroupBlacklist string `json:"sub_group_blacklist" gorm:"type:text" description:"字幕组黑名单"`
}

// TableName 指定表名
func (GlobalSettings) TableName() string {
	return "global_settings"
}

// GetGlobalSettings 获取全局设置
func GetGlobalSettings() (*GlobalSettings, error) {
	var settings GlobalSettings
	result := DB.First(&settings)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// 如果记录不存在，创建默认设置
			settings = GlobalSettings{
				GlobalKeywords:    "",
				ExcludeKeywords:   "",
				SubGroupBlacklist: "",
			}
			if err := DB.Create(&settings).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, result.Error
		}
	}
	return &settings, nil
}

// UpdateGlobalSettings 更新全局设置
func UpdateGlobalSettings(settings *GlobalSettings) error {
	return DB.Save(settings).Error
}
