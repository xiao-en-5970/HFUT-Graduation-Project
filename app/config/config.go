package config

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"

	"github.com/fsnotify/fsnotify"
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

	LogLevel           string
	LogEncoding        string
	LogStacktraceLevel string // 栈追踪起始级别：debug/info/warn/error，默认 error

	JWTSecret     string
	JWTExpireHour int

	OSSRoot           string // OSS 存储根路径，容器内 /oss，对应宿主机 /var/oss
	OSSHost           string // OSS 对外访问域名，如 http://api.xiaoen.xyz，用于返回完整 URL 给前端
	OSSSmallImageSize int    // 压缩图最大边长（像素），如 720 或 540，0 表示不生成压缩图
	OSSSmallImageKB   int    // 压缩图体积上限（KB），如 200，0 表示 200

	// 聚合搜索排序权重（热度 = (收藏*W_C+点赞*W_L+浏览*W_V)×互动衰减）
	SearchWeightCollect        int     // 收藏权重，默认 10
	SearchWeightLike           int     // 点赞权重，默认 5
	SearchWeightView           int     // 浏览权重，默认 1
	SearchInteractionDecayDays float64 // 互动分衰减半衰期（天），默认 90；互动分*=1/(1+距今天数/此值)，0=不衰减
	SearchCombinedRelevance    float64 // combined 排序：相关度系数，默认 100
	SearchCombinedPopularity   float64 // combined 排序：热度系数，默认 0.01

	// 高德地图 Web 服务 Key（用于订单发货/收货地址间步行距离等），见 https://console.amap.com/
	AmapKey string
)

const defaultEnvPath = "/.env"

// LoadConfig 从宿主机固定路径 /.env 或环境变量加载配置
// 服务器：/.env（宿主机路径）；容器：由 docker --env-file 注入；本地：run.sh 手动 export
func LoadConfig() error {
	return LoadConfigFrom(defaultEnvPath)
}

// LoadConfigFrom 从指定路径加载配置
func LoadConfigFrom(path string) error {
	if path == "" {
		path = defaultEnvPath
	}
	if err := godotenv.Load(path); err == nil {
		log.Printf("Loaded config from %s", path)
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
	LogStacktraceLevel = getEnv("LOG_STACKTRACE_LEVEL", "error")

	JWTSecret = getEnv("JWT_SECRET", "your-secret-key-change-in-production")
	JWTExpireHour = getEnvInt("JWT_EXPIRE_HOUR", 24)
	OSSRoot = getEnv("OSS_ROOT", "/oss")
	OSSHost = getEnv("OSS_HOST", "")
	OSSSmallImageSize = getEnvInt("OSS_SMALL_IMAGE_SIZE", 720)
	OSSSmallImageKB = getEnvInt("OSS_SMALL_IMAGE_KB", 200)
	if OSSSmallImageKB <= 0 {
		OSSSmallImageKB = 200
	}

	SearchWeightCollect = getEnvInt("SEARCH_WEIGHT_COLLECT", 10)
	SearchWeightLike = getEnvInt("SEARCH_WEIGHT_LIKE", 5)
	SearchWeightView = getEnvInt("SEARCH_WEIGHT_VIEW", 1)
	SearchInteractionDecayDays = getEnvFloat("SEARCH_INTERACTION_DECAY_DAYS", 90)
	SearchCombinedRelevance = getEnvFloat("SEARCH_COMBINED_RELEVANCE", 100)
	SearchCombinedPopularity = getEnvFloat("SEARCH_COMBINED_POPULARITY", 0.01)

	AmapKey = getEnv("AMAP_KEY", "")

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

func getEnvFloat(key string, defaultValue float64) float64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			return n
		}
	}
	return defaultValue
}

// WatchAndReload 监听 .env 文件变化并热重载配置（仅更新内存变量，DB/Redis 连接需重启才能生效）
func WatchAndReload(envPath string) {
	if envPath == "" {
		envPath = defaultEnvPath
	}
	absPath, err := filepath.Abs(envPath)
	if err != nil {
		log.Printf("config watch: invalid path %s: %v", envPath, err)
		return
	}
	if _, err := os.Stat(absPath); err != nil {
		log.Printf("config watch: file not found %s, skip watching", absPath)
		return
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("config watch: failed to create watcher: %v", err)
		return
	}
	defer watcher.Close()
	if err := watcher.Add(absPath); err != nil {
		log.Printf("config watch: failed to watch %s: %v", absPath, err)
		return
	}
	log.Printf("config watch: watching %s for changes", absPath)
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				if err := LoadConfigFrom(absPath); err != nil {
					log.Printf("config watch: reload failed: %v", err)
				} else {
					log.Printf("config watch: reloaded from %s", absPath)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("config watch error: %v", err)
		}
	}
}

// SetupReloadOnSIGHUP 监听 SIGHUP 信号，收到时重载配置。可通过 kill -HUP <pid> 触发（仅 Unix）
func SetupReloadOnSIGHUP(envPath string) {
	if runtime.GOOS == "windows" {
		return
	}
	if envPath == "" {
		envPath = defaultEnvPath
	}
	path := envPath
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP)
	go func() {
		for range sigChan {
			if err := LoadConfigFrom(path); err != nil {
				log.Printf("config reload (SIGHUP): %v", err)
			} else {
				log.Printf("config reloaded on SIGHUP")
			}
		}
	}()
	log.Printf("config: SIGHUP handler registered (kill -HUP <pid> to reload)")
}
