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

		

		// 公开路由
		v1public := v1.Group("")
		{
			// 内测模式状态检查路由（公开）
			v1public.GET("/beta/status", betaModeController.GetBetaModeStatus)

			
			// 统计和排名相关路由
			v1public.GET("/bangumi/stats/views", controllers.GetBangumiViewStats)         // 番剧播放量统计
			v1public.GET("/bangumi/stats/favorites", controllers.GetBangumiFavoriteStats) // 番剧收藏量统计
			v1public.GET("/bangumi/stats/ratings", controllers.GetBangumiRatingStats)     // 番剧评分统计
			v1public.GET("/bangumi/stats/rankings", controllers.GetBangumiRankings)       // 番剧排行榜
				
			v1public.GET("/bangumi/year/:year", controllers.GetBangumiByYear) // 按年份查询番剧
				
			// 所有番剧相关路由
			v1public.GET("/bangumi", controllers.GetAllBangumi)         // 获取所有番剧
			v1public.GET("/bangumi/stats", controllers.GetBangumiStats) // 番剧统计
			v1public.GET("/bangumi/years", controllers.GetBangumiYears) // 获取所有年份
			v1public.GET("/carousels", carouselController.GetCarousels) // 获取轮播图
			
			v1beta :=v1public.Group("")
			v1beta.Use(middleware.BetaModeMiddleware())
			{
				// 公开但是要内测权限
				v1beta.POST("/register", authController.Register)

			}
			

		}
		v1.POST("/login", authController.Login)

		// 需要登录的路由组
		authenticated := v1.Group("")
		authenticated.Use(middleware.AuthMiddleware())
		{
			// 用户信息路由（不受内测模式限制）
			authenticated.GET("/user/info", authController.GetUserInfo)        // 获取用户信息
			authenticated.PUT("/user/info", authController.UpdateUserInfo)     // 更新用户信息
			authenticated.PUT("/user/password", authController.UpdatePassword) // 更新密码
			authenticated.GET("/user/favorites", controllers.GetUserFavorites) // 获取用户收藏

			// 历史记录
			authenticated.GET("/history/play_history", controllers.GetPlayHistory)           // 获取播放历史
			authenticated.POST("/history/play_history", controllers.AddOrUpdatePlayHistroy)  // 更新播放历史
			authenticated.DELETE("/history/:id/play_history", controllers.DeletePlayHistroy) // 更新播放历史

			// 需要内测权限的路由组
			beta := authenticated.Group("")
			beta.Use(middleware.BetaModeMiddleware())
			{

				// 番剧统计相关路由
				beta.POST("/bangumi/:id/view", controllers.IncrementViewCount)   // 增加番剧播放量
				beta.POST("/bangumi/:id/favorite", controllers.ToggleFavorite)   // 切换番剧收藏状态
				beta.POST("/bangumi/:id/rating", controllers.AddOrUpdateRating)  // 更新或添加用户评分
				beta.DELETE("/bangumi/:id/rating", controllers.DeleteUserRating) // 删除用户评分

				beta.GET("/bangumi/:id/rating", controllers.GetUserRating)                   // 获取用户评分
				beta.GET("/bangumi/:id/stats", controllers.GetBangumiStatsByID)              // 获取番剧统计信息
				beta.GET("/bangumi/:id/rating_stats", controllers.GetBangumiRatingStatsByID) // 获取番剧评分统计信息

				// Bangumi 相关路由

				beta.GET("/bangumi/search", controllers.SearchBangumi)                        // 搜索番剧
				beta.GET("/bangumi/:id", controllers.GetBangumiByID)                          // 获取番剧详情
				beta.GET("/bangumi/grouped_items/:id", controllers.GetGroupedBangumiRSSItems) // 获取番剧组
				beta.GET("/bangumi/items/:id", controllers.GetBangumiRSSItems)                // 获取番剧RSS
				beta.GET("/bangumi/:id/group_episode", controllers.GetGroupEpisodeInfo)       // 获取番剧集数信息
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
