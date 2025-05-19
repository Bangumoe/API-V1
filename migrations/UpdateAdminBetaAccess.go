package migrations

import (
	"backend/models"
	"log"
)

func UpdateAdminBetaAccess() {
	// 更新所有管理员的IsAllowed字段为true
	result := models.DB.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Update("is_allowed", true)
	if result.Error != nil {
		log.Printf("更新管理员内测访问权限失败: %v", result.Error)
		return
	}

	log.Printf("成功更新了 %d 个管理员的内测访问权限", result.RowsAffected)
}
