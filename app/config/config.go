package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var (
	ServerHost string
	ServerPort string
	ServerMode string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	DBTimezone string

	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       string

	LogLevel    string
	LogEncoding string

	JWTSecret     string
	JWTExpireHour int
)

// LoadConfig 从 .env 或环境变量加载配置
// 优先尝试 /opt/app/.env（服务器），其次 .env（本地）
func LoadConfig() error {
	paths := []string{"/opt/app/.env", ".env"}
	for _, p := range paths {
		if err := godotenv.Load(p); err == nil {
			log.Printf("Loaded config from %s", p)
			break
		}
	}

	ServerHost = getEnv("SERVER_HOST", "0.0.0.0")
	ServerPort = getEnv("SERVER_PORT", "8080")
	ServerMode = getEnv("SERVER_MODE", "debug")

	DBHost = getEnv("DB_HOST", "localhost")
	DBPort = getEnv("DB_PORT", "5432")
	DBUser = getEnv("DB_USER", "postgres")
	DBPassword = getEnv("DB_PASSWORD", "postgres")
	DBName = getEnv("DB_NAME", "graduation_project")
	DBSSLMode = getEnv("DB_SSLMODE", "disable")
	DBTimezone = getEnv("DB_TIMEZONE", "PRC")

	RedisHost = getEnv("REDIS_HOST", "localhost")
	RedisPort = getEnv("REDIS_PORT", "6379")
	RedisPassword = getEnv("REDIS_PASSWORD", "")
	RedisDB = getEnv("REDIS_DB", "0")

	LogLevel = getEnv("LOG_LEVEL", "info")
	LogEncoding = getEnv("LOG_ENCODING", "console")

	JWTSecret = getEnv("JWT_SECRET", "your-secret-key-change-in-production")
	JWTExpireHour = getEnvInt("JWT_EXPIRE_HOUR", 24)

	return nil
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultValue
}
