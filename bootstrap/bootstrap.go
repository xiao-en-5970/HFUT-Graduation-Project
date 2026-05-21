package bootstrap

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/router"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/scheduler"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/botinternal"
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

	// 初始化 bot internal API 客户端（QQ 绑定、孤儿账号转发等反向调用 bot 用）
	// BOT_INTERNAL_API_URL/SECRET 缺一个 → botinternal.Default 保持 nil，service 层调用拒绝
	// URL 拼写错（缺 http:// 等）→ Init 返错，但不阻止 hfut 启动（QQ 绑定外其它功能不受影响）
	if berr := botinternal.Init(); berr != nil {
		logger.Error(ctx, "bot internal API client init 失败", zap.Error(berr))
	}
	if botinternal.Default == nil {
		logger.Warnf(ctx, "BOT_INTERNAL_API_URL/SECRET 未完整配置或 URL 不合法，QQ 绑定流程不可用（识别+上架等不受影响）")
	} else {
		logger.Infof(ctx, "bot internal API client initialized successfully")
	}

	// Initialize Gin service
	if err := service.Init(); err != nil {
		logger.Error(ctx, "Failed to initialize Gin service", zap.Error(err))
		return err
	}
	if service.Engine == nil {
		return fmt.Errorf("gin Engine is nil after service.Init")
	}
	logger.Infof(ctx, "Gin service initialized successfully")

	// Setup routes（必须在 Run 之前完成；与 Run 使用的是同一个 *gin.Engine 指针）
	router.SetupRouter(service.Engine)
	routes := service.Engine.Routes()
	if len(routes) == 0 {
		return fmt.Errorf("gin has zero routes after router.SetupRouter — route registration failed")
	}
	logger.Info(ctx, "Gin routes registered and bound to engine",
		zap.Int("route_count", len(routes)),
		zap.String("engine_ptr", fmt.Sprintf("%p", service.Engine)),
	)

	// 启动后台任务（非阻塞）：商品到期自动下架，每 5 分钟扫描一次。
	// 进程退出时由 ctx 被 GC（main 阻塞在 service.Run），协程随主进程一起结束。
	scheduler.StartGoodsAutoOffShelf(ctx)

	// 启动指标持久化：每分钟把 hfut + bot 的分钟桶 + bot 事件流 UPSERT 到 PG，
	// 给运维面板的"过去 1h / 24h / 7d 折线图"提供历史数据。详见 metrics_persist.go。
	scheduler.StartMetricsPersister(ctx)

	// 确保 bot_runtime_config 表存在 + 种子默认值；bot 端拉取 GET /api/v1/bot/runtime-config 用。
	// 失败仅 warn，不影响 hfut 主体功能（bot 拉不到会保留 env 兜底）。
	ensureCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	if err := dao.EnsureBotRuntimeConfigSchema(ensureCtx); err != nil {
		logger.Warn(ctx, "ensure bot_runtime_config schema failed", zap.Error(err))
	}
	cancel()

	// gin.Engine.Run → http.ListenAndServe：成功后会一直阻塞，直到进程退出。
	// Gin 在 release 模式下不会在控制台打印监听地址，容易造成「启动后没日志」的误判，故在此显式打出。
	logger.Info(ctx, "Starting HTTP server (blocking)",
		zap.String("addr", config.ServerHost+":"+config.ServerPort),
		zap.String("server_mode", config.ServerMode),
	)
	logger.Infof(ctx,
		"Ops: smoke test → curl -s http://127.0.0.1:%s/ | jq .  (expect code=200, data.apiPrefix=/api/v1); curl -s http://127.0.0.1:%s/health",
		config.ServerPort, config.ServerPort,
	)
	logger.Infof(ctx,
		"If all APIs fail: check path prefix /api/v1, Docker -p host:container port match, firewall, and reverse-proxy upstream.",
	)
	if err := service.Run(); err != nil {
		logger.Fatal(ctx, "HTTP server exited with error", zap.Error(err))
		return err
	}
	logger.Info(ctx, "HTTP server stopped normally")
	return nil
}
