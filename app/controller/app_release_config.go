package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// AppReleaseInfoURL GET /api/v1/app/release-info-url —— 公开接口，无需 JWT。
//
// 返回 app 内更新功能的元信息 JSON 地址（latest.json 的 URL）。
// 前端 src/api/appUpdate.ts 启动时先调本接口，再去 fetch 拿到的 URL 解析 JSON——
// 这样换 OSS 域名 / 切到 GitHub Release / 紧急下线"更新弹窗"功能 都不需要重新发版前端，
// 改后端环境变量 APP_RELEASE_INFO_URL 重启即可。
//
// 返回体：
//
//	{ "url": "https://oss.xiaoen.xyz/app-release/android/latest.json" }
//
// 若 url 为空字符串，前端按"功能未启用"处理（不弹更新弹窗，不报错）。
func AppReleaseInfoURL(ctx *gin.Context) {
	reply.ReplyOKWithData(ctx, gin.H{
		"url": config.AppReleaseInfoURL,
	})
}
