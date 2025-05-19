package middleware

import (
	"backend/config"
	"backend/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func BetaModeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := config.GetConfig()

		// 如果不是内测模式，直接放行
		if !cfg.IsBetaMode {
			c.Next()
			return
		}

		// 检查用户是否已登录
		claims, exists := c.Get("claims")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":        "内测模式下需要登录",
				"is_beta_mode": true,
			})
			c.Abort()
			return
		}

		// 从数据库中获取最新的用户信息
		userClaims, ok := claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":        "无效的令牌信息",
				"is_beta_mode": true,
			})
			c.Abort()
			return
		}

		userID := uint(userClaims["user_id"].(float64))

		var user models.User
		if err := models.DB.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":        "用户信息获取失败",
				"is_beta_mode": true,
			})
			c.Abort()
			return
		}

		// 检查用户是否被允许访问
		if !user.IsAllowed {
			c.JSON(http.StatusForbidden, gin.H{
				"error":        "您暂无权限访问内测版本",
				"is_beta_mode": true,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
