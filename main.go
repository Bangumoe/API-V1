package main

import (
	"backend/config"
	"backend/controllers"
	_ "backend/docs" // 导入 swagger 生成的文档
	"backend/middleware"
	"backend/migrations"
	"backend/models"
	"backend/services/activity"
	"backend/services/rss"
	"backend/utils"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           动画网站 API
// @version         1.2.0
// @description     这是一个动画网站的后端API服务
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8081
// @BasePath  /api/v1

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description 请在此输入 Bearer token
func main() {
	// 初始化日志系统
	if err := utils.InitLogger(); err != nil {
		log.Fatal("Error initializing logger:", err)
	}

	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	db, err := config.InitDB()
	if err != nil {
		log.Fatal("Error connecting to database:", err)
	}

	// 设置全局数据库连接
	models.SetDB(db)

	// 运行数据库迁移
	migrations.UpdateAdminBetaAccess()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 配置 CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"X-Requested-With",
			"Accept",
			"Accept-Encoding",
			"Accept-Language",
			"Cache-Control",
			"Connection",
			"Host",
			"Pragma",
			"Referer",
			"User-Agent",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"Content-Type",
			"Authorization",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
		AllowWildcard:    true,
	}))

	// 添加静态文件服务
	r.Static("/uploads", "./uploads")

	// 添加 swagger 路由
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 初始化各种服务
	activityService := activity.NewActivityService(db)

	// 初始化控制器时注入 activityService
	authController := controllers.NewAuthController(db, activityService)
	userManagementController := controllers.NewUserManagementController(db)
	carouselController := controllers.NewCarouselController(db)
	betaModeController := controllers.NewBetaModeController(db)

	// 初始化活动记录服务
	activityController := controllers.NewActivityController(activityService)

	// API v1 路由组
	v1 := r.Group("/api/v1")
	{
		// 内测模式状态检查路由（公开）
		v1.GET("/beta/status", betaModeController.GetBetaModeStatus)

		// 公开路由
		v1beta := v1.Group("")
		v1beta.Use(middleware.BetaModeMiddleware())
		{
			v1beta.POST("/register", authController.Register)
		}
		v1.POST("/login", authController.Login)

		// 需要登录的路由组
		authenticated := v1.Group("")
		authenticated.Use(middleware.AuthMiddleware())
		{
			// 用户信息路由（不受内测模式限制）
			authenticated.GET("/user/info", authController.GetUserInfo)
			authenticated.PUT("/user/info", authController.UpdateUserInfo)
			authenticated.PUT("/user/password", authController.UpdatePassword)

			// 需要内测权限的路由组
			beta := authenticated.Group("")
			beta.Use(middleware.BetaModeMiddleware())
			{
				v1.GET("/bangumi/years", controllers.GetBangumiYears)
				v1.GET("/carousels", carouselController.GetCarousels)
				// 番剧统计相关路由
				beta.POST("/bangumi/:id/view", controllers.IncrementViewCount)
				beta.POST("/bangumi/:id/favorite", controllers.ToggleFavorite)
				beta.POST("/bangumi/:id/rating", controllers.AddOrUpdateRating)
				beta.GET("/bangumi/:id/rating", controllers.GetUserRating)
				beta.DELETE("/bangumi/:id/rating", controllers.DeleteUserRating)

				// 统计和排名相关路由
				beta.GET("/bangumi/stats/views", controllers.GetBangumiViewStats)
				beta.GET("/bangumi/stats/favorites", controllers.GetBangumiFavoriteStats)
				beta.GET("/bangumi/stats/ratings", controllers.GetBangumiRatingStats)
				beta.GET("/bangumi/stats/rankings", controllers.GetBangumiRankings)

				// 指定番剧统计相关路由
				beta.GET("/bangumi/:id/stats", controllers.GetBangumiStatsByID)
				beta.GET("/bangumi/:id/rating_stats", controllers.GetBangumiRatingStatsByID)
				beta.GET("/bangumi/year/:year", controllers.GetBangumiByYear)

				// Bangumi 相关路由
				beta.GET("/bangumi", controllers.GetAllBangumi)
				beta.GET("/bangumi/search", controllers.SearchBangumi)
				beta.GET("/bangumi/stats", controllers.GetBangumiStats)
				beta.GET("/bangumi/:id", controllers.GetBangumiByID)
				beta.GET("/bangumi/grouped_items/:id", controllers.GetGroupedBangumiRSSItems)
				beta.GET("/bangumi/items/:id", controllers.GetBangumiRSSItems)
				beta.GET("/bangumi/:id/group_episode", controllers.GetGroupEpisodeInfo)
			}

			// 管理员路由组
			admin := authenticated.Group("/admin")
			admin.Use(middleware.RequireRoles(models.RoleAdmin))
			{
				// 全局设置路由
				admin.GET("/settings", controllers.GetGlobalSettings)
				admin.PUT("/settings", controllers.UpdateGlobalSettings)

				// 内测模式管理路由
				admin.POST("/beta/toggle", betaModeController.ToggleBetaMode)
				admin.POST("/beta/user-access", betaModeController.UpdateUserBetaAccess)

				// 用户管理路由
				admin.GET("/users", userManagementController.GetAllUsers)
				admin.GET("/users/:id", userManagementController.GetUser)
				admin.PUT("/users/:id", userManagementController.UpdateUser)
				admin.DELETE("/users/:id", userManagementController.DeleteUser)

				// 番剧管理路由
				admin.DELETE("/bangumi/:id", controllers.DeleteBangumi)
				admin.PUT("/bangumi/:id", controllers.UpdateBangumi)

				// 系统统计和状态路由
				admin.GET("/stats", controllers.GetSystemStats)
				admin.GET("/system/status", controllers.GetSystemStatus)
				admin.GET("/logs", controllers.GetLogs)

				// RSSFeed相关API
				v1.GET("/rss_feeds", controllers.GetAllRSSFeeds)
				v1.GET("/rss_feeds/:id", controllers.GetRSSFeedByID)
				v1.POST("/rss_feeds", controllers.CreateRSSFeed)
				v1.PUT("/rss_feeds/:id", controllers.UpdateRSSFeed)
				v1.DELETE("/rss_feeds/:id", controllers.DeleteRSSFeed)
				v1.POST("/rss_feeds/update", controllers.ManualUpdateRSSFeeds)
				v1.POST("/rss_feeds/:id/update", controllers.UpdateRSSFeedByID)

				// 活动记录路由
				admin.GET("/activities", activityController.GetRecentActivities)

				// Carousel管理路由
				admin.POST("/carousels", carouselController.CreateCarousel)
				admin.GET("/carousels/:id", carouselController.GetCarousel)
				admin.PUT("/carousels/:id", carouselController.UpdateCarousel)
				admin.DELETE("/carousels/:id", carouselController.DeleteCarousel)
				admin.PUT("/carousels/order", carouselController.UpdateCarouselOrder)
			}

		}
		// 单独注册WebSocket日志路由，不加任何中间件
		v1.GET("/admin/logs/watch", controllers.WatchLogs)
	}

	// 初始化RSS定时任务调度器
	rssScheduler := rss.NewRSSUpdateScheduler(db)
	go rssScheduler.Start()

	r.Run(":8081")
}
