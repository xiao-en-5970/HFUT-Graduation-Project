package bootstrap

import (
	"context"

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

	// Start the server (this will block)
	logger.Infof(ctx, "Starting server...")
	if err := service.Run(); err != nil {
		logger.Fatal(ctx, "Failed to start server", zap.Error(err))
		return err
	}
	return nil
}
