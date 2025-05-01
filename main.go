package main

import (
	"backend/config"
	"backend/controllers"
	_ "backend/docs" // 导入 swagger 生成的文档
	"backend/middleware"
	"backend/models"
	"backend/services/activity"
	"backend/services/rss"
	"backend/utils"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           动画网站 API
// @version         1.0
// @description     这是一个动画网站的后端API服务
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

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

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 添加静态文件服务
	r.Static("/uploads", "./uploads")

	// 添加 swagger 路由
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 初始化各种服务
	activityService := activity.NewActivityService(db)

	// 初始化控制器时注入 activityService
	authController := controllers.NewAuthController(db, activityService)
	userManagementController := controllers.NewUserManagementController(db)

	// 初始化活动记录服务
	activityController := controllers.NewActivityController(activityService)

	// API v1 路由组
	v1 := r.Group("/api/v1")
	{
		// 公开路由
		v1.POST("/register", authController.Register)
		v1.POST("/login", authController.Login)

		// RSS相关路由 - 需要登录但不需要管理员权限
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/user/info", authController.GetUserInfo)

			// 添加 Bangumi 相关路由
			protected.GET("/bangumi", controllers.GetAllBangumi)
			protected.GET("/bangumi/search", controllers.SearchBangumi)
			protected.GET("/bangumi/stats", controllers.GetBangumiStats)
			protected.GET("/bangumi/:id", controllers.GetBangumiByID)
			protected.GET("/bangumi/grouped_items/:id", controllers.GetGroupedBangumiRSSItems)

			// 添加新的RSS条目查询路由
			protected.GET("/bangumi/items/:id", controllers.GetBangumiRSSItems)
			protected.GET("/bangumi/:id/group_episode", controllers.GetGroupEpisodeInfo)
		}

		// 管理员路由组
		admin := v1.Group("/admin")
		admin.Use(middleware.AuthMiddleware())
		admin.Use(middleware.RequireRoles(models.RoleAdmin))
		{
			// 用户管理路由
			admin.GET("/users", userManagementController.GetAllUsers)
			admin.GET("/users/:id", userManagementController.GetUser)
			admin.PUT("/users/:id", userManagementController.UpdateUser)
			admin.DELETE("/users/:id", userManagementController.DeleteUser)

			// 添加系统统计和状态路由
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
			v1.POST("/rss_feeds/:id/update", controllers.UpdateRSSFeedByID) // 添加新的路由
			// 添加活动记录路由
			v1.GET("/activities", activityController.GetRecentActivities)
		}

		// ！！！单独注册WebSocket日志路由，不加任何中间件！！！
		v1.GET("/admin/logs/watch", controllers.WatchLogs)
	}

	// 初始化RSS定时任务调度器
	rssScheduler := rss.NewRSSUpdateScheduler(db)
	go rssScheduler.Start()

	r.Run(":8081")
}
