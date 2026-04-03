package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// MapConfig GET /config/map 返回高德 JS Key + 安全密钥（加载地图脚本前需设置 _AMapSecurityConfig）
func MapConfig(ctx *gin.Context) {
	reply.ReplyOKWithData(ctx, gin.H{
		"amap_web_key":           config.AmapWebKey,
		"amap_security_js_code": config.AmapWebSecurityCode,
	})
}
