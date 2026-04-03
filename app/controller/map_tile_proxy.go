package controller

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
)

var mapTileProxyClient = &http.Client{Timeout: 45 * time.Second}

// upstreamMartinBase 从 MAP_TILES_URL 解析 Martin 基路径（不含 {z}/{x}/{y}），供服务端拉瓦片
func upstreamMartinBase(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if i := strings.Index(s, "{z}"); i > 0 {
		return strings.TrimRight(s[:i], "/")
	}
	return strings.TrimRight(s, "/")
}

// MapTileProxy GET /map/tiles/:z/:x/:y 反向代理至 Martin；浏览器只请求本 API，不直连瓦片机
func MapTileProxy(c *gin.Context) {
	base := upstreamMartinBase(config.MapTilesURL)
	if base == "" {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	z := c.Param("z")
	x := c.Param("x")
	y := c.Param("y")
	target := base + "/" + z + "/" + x + "/" + y
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, target, nil)
	if err != nil {
		c.AbortWithStatus(http.StatusBadGateway)
		return
	}
	resp, err := mapTileProxyClient.Do(req)
	if err != nil {
		c.AbortWithStatus(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for _, h := range []string{"Content-Type", "Content-Encoding", "Cache-Control"} {
		if v := resp.Header.Get(h); v != "" {
			c.Writer.Header().Set(h, v)
		}
	}
	c.Status(resp.StatusCode)
	_, _ = io.Copy(c.Writer, resp.Body)
}
