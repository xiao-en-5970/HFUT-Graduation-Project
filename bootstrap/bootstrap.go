package bootstrap

import (
	"context"
	"os"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/router"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/logger"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/redis"
	"go.uber.org/zap"
)

// Boot initializes all components
func Boot() error {
	// Load configuration
	ctx := context.Background()
	if err := config.LoadConfig(); err != nil {
		return err
	}
	// 可选：CONFIG_WATCH=1 启用 .env 文件监听热重载；SIGHUP 信号也可触发重载
	if os.Getenv("CONFIG_WATCH") == "1" {
		go config.WatchAndReload("/.env")
	}
	config.SetupReloadOnSIGHUP("/.env")

	// Initialize logger first (needed for other components)
	if err := logger.Init(); err != nil {
		return err
	}
	defer logger.Sync()

	logger.Infof(ctx, "Logger initialized successfully")

	// Initialize PostgreSQL
	if err := pgsql.Init(); err != nil {
		logger.Error(ctx, "Failed to initialize PostgreSQL", zap.Error(err))
		return err
	}
	logger.Infof(ctx, "PostgreSQL initialized successfully")

	// Initialize Redis
	if err := redis.Init(); err != nil {
		logger.Error(ctx, "Failed to initialize Redis", zap.Error(err))
		return err
	}
	logger.Infof(ctx, "Redis initialized successfully")

	// Initialize Gin service
	if err := service.Init(); err != nil {
		logger.Error(ctx, "Failed to initialize Gin service", zap.Error(err))
		return err
	}
	logger.Infof(ctx, "Gin service initialized successfully")

	// Setup routes
	router.SetupRouter(service.Engine)
	logger.Infof(ctx, "Routes initialized successfully")

	// gin.Engine.Run → http.ListenAndServe：成功后会一直阻塞，直到进程退出。
	// Gin 在 release 模式下不会在控制台打印监听地址，容易造成「启动后没日志」的误判，故在此显式打出。
	logger.Info(ctx, "Starting HTTP server (blocking)",
		zap.String("addr", config.ServerHost+":"+config.ServerPort),
		zap.String("server_mode", config.ServerMode),
	)
	if err := service.Run(); err != nil {
		logger.Fatal(ctx, "HTTP server exited with error", zap.Error(err))
		return err
	}
	logger.Info(ctx, "HTTP server stopped normally")
	return nil
}
