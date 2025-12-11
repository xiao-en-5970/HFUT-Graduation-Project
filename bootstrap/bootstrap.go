package bootstrap

import (
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/logger"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/redis"
	"go.uber.org/zap"
)

// Boot initializes all components
func Boot() error {
	// Load configuration
	if err := config.LoadConfig(); err != nil {
		return err
	}

	// Initialize logger first (needed for other components)
	if err := logger.Init(); err != nil {
		return err
	}
	defer logger.Sync()

	logger.Logger.Info("Logger initialized successfully")

	// Initialize PostgreSQL
	if err := pgsql.Init(); err != nil {
		logger.Logger.Error("Failed to initialize PostgreSQL", zap.Error(err))
		return err
	}
	logger.Logger.Info("PostgreSQL initialized successfully")

	// Initialize Redis
	if err := redis.Init(); err != nil {
		logger.Logger.Error("Failed to initialize Redis", zap.Error(err))
		return err
	}
	logger.Logger.Info("Redis initialized successfully")

	// Initialize Gin service
	if err := service.Init(); err != nil {
		logger.Logger.Error("Failed to initialize Gin service", zap.Error(err))
		return err
	}
	logger.Logger.Info("Gin service initialized successfully")

	// Start the server (this will block)
	logger.Logger.Info("Starting server...")
	if err := service.Run(); err != nil {
		logger.Logger.Fatal("Failed to start server", zap.Error(err))
		return err
	}

	return nil
}
