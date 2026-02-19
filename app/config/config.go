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

	OSSRoot string // OSS 存储根路径，容器内 /oss，对应宿主机 /var/oss
	OSSHost string // OSS 对外访问域名，如 http://api.xiaoen.xyz，用于返回完整 URL 给前端
)

// LoadConfig 从宿主机固定路径 /.env 或环境变量加载配置
// 服务器：/.env（宿主机路径）；容器：由 docker --env-file 注入；本地：run.sh 手动 export
func LoadConfig() error {
	if err := godotenv.Load("/.env"); err == nil {
		log.Printf("Loaded config from /.env")
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
	DBTimezone = getEnv("DB_TIMEZONE", "Asia/Shanghai")

	RedisHost = getEnv("REDIS_HOST", "localhost")
	RedisPort = getEnv("REDIS_PORT", "6379")
	RedisPassword = getEnv("REDIS_PASSWORD", "")
	RedisDB = getEnv("REDIS_DB", "0")

	LogLevel = getEnv("LOG_LEVEL", "info")
	LogEncoding = getEnv("LOG_ENCODING", "console")

	JWTSecret = getEnv("JWT_SECRET", "your-secret-key-change-in-production")
	JWTExpireHour = getEnvInt("JWT_EXPIRE_HOUR", 24)
	OSSRoot = getEnv("OSS_ROOT", "/oss")
	OSSHost = getEnv("OSS_HOST", "")

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
