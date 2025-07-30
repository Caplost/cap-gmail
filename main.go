package main

import (
	"gmail-forwarding-system/config"
	"gmail-forwarding-system/database"
	"gmail-forwarding-system/handlers"
	"gmail-forwarding-system/middlewares"
	"gmail-forwarding-system/services"
	"gmail-forwarding-system/utils"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化日志
	utils.InitLogger()
	utils.LogInfo("启动邮件转发系统...")

	// 加载配置
	config.LoadConfig()

	// 初始化数据库
	database.InitDatabase()
	db := database.GetDB()

	// 创建邮件监听服务
	var emailMonitor *services.EmailMonitor
	if config.AppConfig.AutoMonitor {
		var err error
		emailMonitor, err = services.NewEmailMonitor(db)
		if err != nil {
			utils.LogError("创建邮件监听服务失败", err)
		} else {
			utils.LogInfo("✅ 邮件监听服务创建成功")
		}
	}

	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)

	// 创建Gin引擎
	r := gin.New()

	// 添加中间件
	r.Use(middlewares.CORSMiddleware())
	r.Use(middlewares.LoggerMiddleware())
	r.Use(gin.Recovery())

	// 创建处理器
	targetHandler := handlers.NewTargetHandler(db)
	emailHandler, err := handlers.NewEmailHandler(db)
	if err != nil {
		log.Fatal("创建邮件处理器失败:", err)
	}

	// 设置路由
	setupRoutes(r, targetHandler, emailHandler, emailMonitor)

	// 启动邮件监听服务 (在Web服务之前启动)
	if emailMonitor != nil && config.AppConfig.AutoMonitor {
		utils.LogInfo("🔄 启动自动邮件监听...")
		emailMonitor.Start()
	}

	// 设置优雅关闭
	setupGracefulShutdown(emailMonitor)

	// 启动服务器
	port := ":" + config.AppConfig.ServerPort
	utils.LogInfo("🌐 Web服务器启动在端口 " + port)
	utils.LogInfo("📋 API文档: http://localhost" + port + "/health")

	if emailMonitor != nil {
		utils.LogInfo("📧 邮件监听: 已启用 (间隔: " + string(rune(config.AppConfig.EmailCheckInterval)) + "秒)")
	} else {
		utils.LogInfo("📧 邮件监听: 已禁用 (手动模式)")
	}

	if err := r.Run(port); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}

// setupRoutes 设置路由
func setupRoutes(r *gin.Engine, targetHandler *handlers.TargetHandler, emailHandler *handlers.EmailHandler, emailMonitor *services.EmailMonitor) {
	// API分组
	api := r.Group("/api")

	// 转发对象管理路由
	targets := api.Group("/targets")
	{
		targets.GET("", targetHandler.GetTargets)
		targets.GET("/:id", targetHandler.GetTarget)
		targets.POST("", targetHandler.CreateTarget)
		targets.PUT("/:id", targetHandler.UpdateTarget)
		targets.DELETE("/:id", targetHandler.DeleteTarget)
	}

	// 邮件处理路由
	emails := api.Group("/emails")
	{
		emails.POST("/process", emailHandler.ProcessEmails)
		emails.GET("/logs", emailHandler.GetEmailLogs)
	}

	// 监听服务管理路由
	if emailMonitor != nil {
		monitor := api.Group("/monitor")
		{
			monitor.GET("/status", func(c *gin.Context) {
				c.JSON(200, emailMonitor.GetStatus())
			})
			monitor.POST("/start", func(c *gin.Context) {
				emailMonitor.Start()
				c.JSON(200, gin.H{"message": "邮件监听已启动"})
			})
			monitor.POST("/stop", func(c *gin.Context) {
				emailMonitor.Stop()
				c.JSON(200, gin.H{"message": "邮件监听已停止"})
			})
		}
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		status := gin.H{
			"status":  "ok",
			"message": "Gmail转发系统运行正常",
			"version": "1.0.0",
			"features": gin.H{
				"auto_monitor": config.AppConfig.AutoMonitor,
				"imap_enabled": config.AppConfig.IMAPUser != "",
				"smtp_enabled": config.AppConfig.SMTPUser != "",
			},
		}

		if emailMonitor != nil {
			status["monitor"] = emailMonitor.GetStatus()
		}

		c.JSON(200, status)
	})
}

// setupGracefulShutdown 设置优雅关闭
func setupGracefulShutdown(emailMonitor *services.EmailMonitor) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		utils.LogInfo("🛑 接收到关闭信号，正在优雅关闭...")

		if emailMonitor != nil {
			emailMonitor.Stop()
		}

		utils.LogInfo("✅ 系统已优雅关闭")
		os.Exit(0)
	}()
}
