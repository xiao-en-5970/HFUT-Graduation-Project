package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/logger"
	"go.uber.org/zap"
)

// metricsMiddleware 是 app/middleware/metrics.go 的 inline 等价物，
// 直接放在 service 包以避免 service ↔ middleware 形成循环 import：
// middleware 已经 import service.Metrics()。
func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		bw := &mwBodyCapture{ResponseWriter: c.Writer, buf: bytes.NewBuffer(nil), max: 1024}
		c.Writer = bw
		c.Next()
		latency := time.Since(start).Milliseconds()
		status := c.Writer.Status()
		biz := 0
		if status == 200 && bw.buf.Len() > 0 {
			var head struct {
				Code int `json:"code"`
			}
			if err := json.Unmarshal(bw.buf.Bytes(), &head); err == nil {
				biz = head.Code
			}
		}
		Metrics().IncRequest(c.Request.Method, c.FullPath(), status, biz, latency)
	}
}

type mwBodyCapture struct {
	gin.ResponseWriter
	buf *bytes.Buffer
	max int
}

func (b *mwBodyCapture) Write(p []byte) (int, error) {
	if b.buf.Len() < b.max {
		room := b.max - b.buf.Len()
		if room >= len(p) {
			b.buf.Write(p)
		} else {
			b.buf.Write(p[:room])
		}
	}
	return b.ResponseWriter.Write(p)
}

func (b *mwBodyCapture) WriteString(s string) (int, error) {
	return b.Write([]byte(s))
}

var Engine *gin.Engine

// Init initializes Gin service
func Init() error {
	// Set Gin mode
	gin.SetMode(config.ServerMode)

	// Create Gin engine
	Engine = gin.New()

	// Add default middleware
	Engine.Use(gin.Logger())
	Engine.Use(gin.Recovery())
	// 运维指标采集中间件——计入 service.Metrics() 单例，供 admin 面板查询
	Engine.Use(metricsMiddleware())

	// Health check endpoint
	Engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	return nil
}

// Run 启动 HTTP 服务。
//
// gin.Engine.Run 内部等价于 http.ListenAndServe，成功后会一直阻塞在 Serve 上直到进程结束，
// 因此「Starting HTTP 之后没新日志」常被误判为卡死；此处拆成 Listen + 日志 + Serve，
// 便于确认：若打出 listener bound，说明端口已绑定成功，随后无输出是正常阻塞而非死锁。
func Run() error {
	if Engine == nil {
		return fmt.Errorf("gin Engine is nil in service.Run — must call service.Init and router.SetupRouter first")
	}
	addr := fmt.Sprintf("%s:%s", config.ServerHost, config.ServerPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	ctx := context.Background()
	logger.Info(ctx, "HTTP listener bound, entering http.Server.Serve (blocks until shutdown)",
		zap.String("listen_addr", ln.Addr().String()),
		zap.String("handler_engine_ptr", fmt.Sprintf("%p", Engine)),
		zap.Int("registered_routes", len(Engine.Routes())),
	)
	srv := &http.Server{
		Handler:           Engine.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
	}
	return srv.Serve(ln)
}
