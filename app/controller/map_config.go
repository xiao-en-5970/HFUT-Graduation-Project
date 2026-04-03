package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// MapConfig GET /config/map 返回高德 JS Key（管理后台/客户端地图选点；与 Web 服务 AMAP_KEY 可分开配置）
func MapConfig(ctx *gin.Context) {
	reply.ReplyOKWithData(ctx, gin.H{"amap_web_key": config.AmapWebKey})
}
