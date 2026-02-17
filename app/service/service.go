package service

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
)

var Engine *gin.Engine

// Init initializes Gin service
func Init() error {
	// Set Gin mode
	gin.SetMode(config.ServerMode())

	// Create Gin engine
	Engine = gin.New()

	// Add default middleware
	Engine.Use(gin.Logger())
	Engine.Use(gin.Recovery())

	// Health check endpoint
	Engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	return nil
}

// Run starts the Gin server
func Run() error {
	addr := fmt.Sprintf("%s:%s", config.ServerHost(), config.ServerPort())
	return Engine.Run(addr)
}
