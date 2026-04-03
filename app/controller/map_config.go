package controller

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// MapConfig GET /config/map 返回经本 API 转发的瓦片 URL 模板（前端不直连 Martin）
func MapConfig(ctx *gin.Context) {
	if upstreamMartinBase(config.MapTilesURL) == "" {
		reply.ReplyOKWithData(ctx, gin.H{
			"map_tiles_url": "",
		})
		return
	}
	public := publicAPIPrefix(ctx) + "/api/v1/map/tiles/{z}/{x}/{y}"
	reply.ReplyOKWithData(ctx, gin.H{
		"map_tiles_url": public,
	})
}

// publicAPIPrefix 用于生成前端可请求的绝对 URL（考虑反向代理头）
func publicAPIPrefix(c *gin.Context) string {
	scheme := "http"
	if xfp := c.GetHeader("X-Forwarded-Proto"); xfp != "" {
		scheme = strings.ToLower(strings.TrimSpace(xfp))
		if scheme != "https" && scheme != "http" {
			scheme = "http"
		}
	} else if c.Request.TLS != nil {
		scheme = "https"
	}
	host := c.Request.Host
	if xf := c.GetHeader("X-Forwarded-Host"); xf != "" {
		host = strings.TrimSpace(xf)
	}
	return scheme + "://" + host
}
