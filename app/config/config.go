package config

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
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

	// 推荐系统（方案 B：标签画像 + 双路召回 + ε-greedy 打散 + refresh_token 稳定分页）
	RecoInterestQuota         float64 // 兴趣池配额，默认 0.6（兴趣 60% / 探索 40%）
	RecoRecentDays            int     // 画像聚合追溯 N 天行为，默认 3650（≈10 年，等同于永久记住喜好；时间衰减由 RecoBehaviorTimeDecayDays 控制）
	RecoProfileTTL            int     // 画像缓存 TTL 秒，默认 600
	RecoSeenTTLDays           int     // seen 集合（已曝光/已浏览）TTL 天，默认 3（短期去重，不影响画像）
	RecoCandidateMultiplier   int     // 候选池放大倍数（兴趣池 & 探索池候选= pageSize × 此倍数），默认 3
	RecoTopTagsLimit          int     // 画像保留的 top 标签数，默认 20
	RecoTopAuthorsLimit       int     // 画像保留的 top 作者数，默认 20
	RecoFreshnessDecayDays    float64 // 新鲜度衰减半衰期（天），默认 14；freshness=1/(1+days/decay)
	RecoInterestKeepTopRatio  float64 // 兴趣池不打散的置顶比例，默认 0.1（高分兴趣始终优先）
	RecoExploreKeepTopRatio   float64 // 探索池不打散的置顶比例，默认 0.3（热门内容在探索槽位始终靠前）
	RecoBehaviorWeightView    float64 // view 动作权重，默认 1
	RecoBehaviorWeightLike    float64 // like 动作权重，默认 5
	RecoBehaviorWeightCollect float64 // collect 动作权重，默认 8
	RecoBehaviorWeightComment float64 // comment 动作权重，默认 3
	RecoBehaviorWeightSearch  float64 // search 动作权重，默认 2
	RecoBehaviorTimeDecayDays float64 // 画像行为的时间半衰期（天），默认 30；近期行为权重更高，老行为不会突然消失

	// Martin 瓦片上游（仅服务端访问，可写 http://127.0.0.1:50001/tiles 或带 {z}/{x}/{y} 的完整模板）
	MapTilesURL string
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

	RecoInterestQuota = getEnvFloat("RECO_INTEREST_QUOTA", 0.6)
	if RecoInterestQuota < 0 || RecoInterestQuota > 1 {
		RecoInterestQuota = 0.6
	}
	// 默认 3650 天（约 10 年），近似"永久记住喜好"；配合时间半衰期实现"老兴趣淡但不消失"
	RecoRecentDays = getEnvInt("RECO_RECENT_DAYS", 3650)
	if RecoRecentDays < 1 {
		RecoRecentDays = 3650
	}
	RecoProfileTTL = getEnvInt("RECO_PROFILE_TTL", 600)
	RecoSeenTTLDays = getEnvInt("RECO_SEEN_TTL_DAYS", 3)
	RecoCandidateMultiplier = getEnvInt("RECO_CANDIDATE_MULTIPLIER", 3)
	if RecoCandidateMultiplier < 1 {
		RecoCandidateMultiplier = 3
	}
	RecoTopTagsLimit = getEnvInt("RECO_TOP_TAGS_LIMIT", 20)
	RecoTopAuthorsLimit = getEnvInt("RECO_TOP_AUTHORS_LIMIT", 20)
	RecoFreshnessDecayDays = getEnvFloat("RECO_FRESHNESS_DECAY_DAYS", 14)
	RecoInterestKeepTopRatio = getEnvFloat("RECO_INTEREST_KEEP_TOP_RATIO", 0.1)
	if RecoInterestKeepTopRatio < 0 || RecoInterestKeepTopRatio > 1 {
		RecoInterestKeepTopRatio = 0.1
	}
	RecoExploreKeepTopRatio = getEnvFloat("RECO_EXPLORE_KEEP_TOP_RATIO", 0.3)
	if RecoExploreKeepTopRatio < 0 || RecoExploreKeepTopRatio > 1 {
		RecoExploreKeepTopRatio = 0.3
	}
	RecoBehaviorWeightView = getEnvFloat("RECO_BEHAVIOR_WEIGHT_VIEW", 1)
	RecoBehaviorWeightLike = getEnvFloat("RECO_BEHAVIOR_WEIGHT_LIKE", 5)
	RecoBehaviorWeightCollect = getEnvFloat("RECO_BEHAVIOR_WEIGHT_COLLECT", 8)
	RecoBehaviorWeightComment = getEnvFloat("RECO_BEHAVIOR_WEIGHT_COMMENT", 3)
	RecoBehaviorWeightSearch = getEnvFloat("RECO_BEHAVIOR_WEIGHT_SEARCH", 2)
	// 默认 30 天：每个月老行为权重减半，近一年前的行为仍有可感知贡献，不会一刀切
	RecoBehaviorTimeDecayDays = getEnvFloat("RECO_BEHAVIOR_TIME_DECAY_DAYS", 30)

	MapTilesURL = strings.TrimSpace(getEnv("MAP_TILES_URL", ""))

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
