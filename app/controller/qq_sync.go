// Package controller 的 qq_sync.go 是"旗下号信息同步"相关的 HTTP 入口。
//
// 路由汇总：
//
//	GET  /api/v1/bot/users/qq-children    bot 用，分页拉所有旗下号（同步任务遍历）
//	GET  /api/v1/bot/qq-sync/pending      bot 用，poll 是否有 admin 触发的强制同步
//	POST /api/v1/admin/qq-children/sync   admin 用，"立即同步"按钮触发一次 bot 全量回填
//
// 数据模型：用 Redis key bot:qq_sync:force_at 存 admin 最后一次触发的 unix-ms 时间戳；
// bot 定期 poll 这个值，发现它比自己上次同步的 last_seen_force_at 新就立即跑一轮。
// 不持久化到 DB —— 这是个"瞬时同步信号"，bot / Redis 任一重启就丢失也无所谓
// （还有 6h 的常规定时同步兜底）。
package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// BotListQQChildren GET /api/v1/bot/users/qq-children?cursor=&limit=
//
// cursor = 上一页最后一个 user_id（默认 0 = 从头）；按 id ASC 翻页，最大 500 / 页。
// 返回字段只保留同步任务需要的：user_id / qq_number / nickname / qq_avatar_url /
// created_in_group_id。普通账号不会出现在结果里。
func BotListQQChildren(ctx *gin.Context) {
	cursor, _ := strconv.ParseUint(ctx.DefaultQuery("cursor", "0"), 10, 32)
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "200"))
	if limit < 1 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	out, nextCursor, err := service.BotListQQChildren(ctx.Request.Context(), uint(cursor), limit)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{
		"list":        out,
		"next_cursor": nextCursor,
		"limit":       limit,
	})
}

// BotQQSyncPending GET /api/v1/bot/qq-sync/pending
//
// 返回当前的 force_at（admin 最近一次按"立即同步"按钮的 unix-ms 时间戳）。
// bot 自己存 last_seen_force_at；poll 出 force_at 比 last_seen 新 → 立即跑一轮全量同步。
func BotQQSyncPending(ctx *gin.Context) {
	at, err := service.QQSyncGetForceAt(ctx.Request.Context())
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"force_at": at})
}

// AdminTriggerQQDisplaySync POST /api/v1/admin/qq-children/sync
//
// 管理后台"立即同步 QQ 头像 / 昵称"按钮。把 force_at 置为 now()，bot 下次 poll
// 时看到就立即跑一轮——异步执行，不等 bot 跑完。
func AdminTriggerQQDisplaySync(ctx *gin.Context) {
	at, err := service.QQSyncSetForceNow(ctx.Request.Context())
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{
		"force_at": at,
		"message":  "已通知 bot 同步；通常 1 分钟内完成",
	})
}
