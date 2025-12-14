package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var (
	// Server config
	ServerHost string
	ServerPort string
	ServerMode string

	// PostgreSQL config
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	DBTimezone string

	// Redis config
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       string

	// Log config
	LogLevel    string
	LogEncoding string

	// JWT config
	JWTSecret     string
	JWTExpireHour int
)

// LoadConfig loads configuration from environment variables
func LoadConfig() error {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Server configuration
	ServerHost = getEnv("SERVER_HOST", "0.0.0.0")
	ServerPort = getEnv("SERVER_PORT", "8080")
	ServerMode = getEnv("SERVER_MODE", "debug")

	// PostgreSQL configuration
	DBHost = getEnv("DB_HOST", "localhost")
	DBPort = getEnv("DB_PORT", "5432")
	DBUser = getEnv("DB_USER", "postgres")
	DBPassword = getEnv("DB_PASSWORD", "postgres")
	DBName = getEnv("DB_NAME", "graduation_project")
	DBSSLMode = getEnv("DB_SSLMODE", "disable")
	// 使用 PRC（PostgreSQL 内置支持）而不是 Asia/Shanghai（需要系统时区数据）
	// 在 Docker 容器中，PRC 更可靠
	DBTimezone = getEnv("DB_TIMEZONE", "PRC")

	// Redis configuration
	RedisHost = getEnv("REDIS_HOST", "localhost")
	RedisPort = getEnv("REDIS_PORT", "6379")
	RedisPassword = getEnv("REDIS_PASSWORD", "")
	RedisDB = getEnv("REDIS_DB", "0")

	// Log configuration
	LogLevel = getEnv("LOG_LEVEL", "info")
	LogEncoding = getEnv("LOG_ENCODING", "console")

	// JWT configuration
	JWTSecret = getEnv("JWT_SECRET", "your-secret-key-change-in-production")
	JWTExpireHour = getEnvInt("JWT_EXPIRE_HOUR", 24)

	return nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an environment variable as int or returns a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
