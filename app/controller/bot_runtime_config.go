// Package controller 的 bot_runtime_config.go：QQ-bot 运行时配置的 HTTP 入口。
//
// 路由汇总：
//
//	GET  /api/v1/bot/runtime-config     bot 拉取全集（启动 + 周期 poll）
//	GET  /api/v1/admin/bot-config       admin UI 列表
//	PUT  /api/v1/admin/bot-config       admin UI 更新单 key 或批量
//
// 数据模型：详见 dao/bot_runtime_config.go 与 package/sql/migrate_bot_runtime_config.sql。
//
// bot 拉到 JSON 后用 atomic.Pointer 热替换内存配置；admin 更新后无需通知 bot，
// 等 bot 下一次 poll（默认 60s）自然生效——跟 qq-sync 同套模式。
package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
	"gorm.io/gorm"
)

// BotRuntimeConfigGet GET /api/v1/bot/runtime-config
//
// bot 端调用。返回扁平 map：{ "auto_reply_whitelist": [...], "ops_group_ids": [...], "silent_mode": false }
// value 字段保持 JSON 原值，由 bot 按 key 解析；未配置的 key 不出现在 map 里。
//
// bot 在拉取失败时应该保留 env / 上次同步的内存值——这里不会主动塞默认，调用方负责兜底。
func BotRuntimeConfigGet(ctx *gin.Context) {
	rows, err := dao.BotRuntimeConfig().List(ctx.Request.Context())
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	out := make(map[string]json.RawMessage, len(rows))
	for _, r := range rows {
		if len(r.Value) == 0 {
			continue
		}
		out[r.Key] = r.Value
	}
	reply.ReplyOKWithData(ctx, out)
}

// AdminBotConfigList GET /api/v1/admin/bot-config
//
// 管理后台列表。比 bot 端那条多了 updated_at / updated_by / comment 元信息，方便 admin
// 看到"上次谁改的、改的什么、字段说明是什么"。
func AdminBotConfigList(ctx *gin.Context) {
	rows, err := dao.BotRuntimeConfig().List(ctx.Request.Context())
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	// 把 jsonb 解码后展平成对前端友好的形态
	items := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		// value 已经是 json.RawMessage；再解一次给前端拿到原生 JSON 类型（不是 string）
		var v interface{}
		if err := json.Unmarshal(r.Value, &v); err != nil {
			v = string(r.Value) // 兜底：解析失败给字符串原文
		}
		items = append(items, gin.H{
			"key":        r.Key,
			"value":      v,
			"updated_at": r.UpdatedAt,
			"updated_by": r.UpdatedBy,
			"comment":    r.Comment,
		})
	}
	reply.ReplyOKWithData(ctx, gin.H{"items": items})
}

// AdminBotConfigUpdate PUT /api/v1/admin/bot-config
//
// 请求体形态（数组）：
//
//	{ "items": [ {"key": "auto_reply_whitelist", "value": [123,456]}, ... ] }
//
// 单 key 单调用也可以——items 数组只放一项即可，避免再加 PUT/:key 路径。
//
// 每一项必须给完整的 value（GORM Upsert 写整列），不支持 patch。
func AdminBotConfigUpdate(ctx *gin.Context) {
	var body struct {
		Items []struct {
			Key   string          `json:"key"`
			Value json.RawMessage `json:"value"`
		} `json:"items"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if len(body.Items) == 0 {
		reply.ReplyInvalidParams(ctx, errors.New("items 不能为空"))
		return
	}
	// 校验 key 合法 + value 是合法 JSON
	allowedKeys := map[string]bool{
		dao.BotCfgAutoReplyWhitelist: true,
		dao.BotCfgOpsGroupIDs:        true,
		dao.BotCfgSilentMode:         true,
	}
	for _, it := range body.Items {
		if !allowedKeys[it.Key] {
			reply.ReplyInvalidParams(ctx, fmt.Errorf("不支持的配置项 key=%q", it.Key))
			return
		}
		if len(it.Value) == 0 {
			reply.ReplyInvalidParams(ctx, fmt.Errorf("key=%s 的 value 不能为空", it.Key))
			return
		}
		if !json.Valid(it.Value) {
			reply.ReplyInvalidParams(ctx, fmt.Errorf("key=%s 的 value 不是合法 JSON", it.Key))
			return
		}
		// 类型语义校验：白名单 / ops 必须是 int64 数组；silent_mode 必须是 bool
		switch it.Key {
		case dao.BotCfgAutoReplyWhitelist, dao.BotCfgOpsGroupIDs:
			var arr []int64
			if err := json.Unmarshal(it.Value, &arr); err != nil {
				reply.ReplyInvalidParams(ctx,
					fmt.Errorf("key=%s 必须是 int64 数组：%v", it.Key, err))
				return
			}
		case dao.BotCfgSilentMode:
			var b bool
			if err := json.Unmarshal(it.Value, &b); err != nil {
				reply.ReplyInvalidParams(ctx,
					fmt.Errorf("key=%s 必须是 bool：%v", it.Key, err))
				return
			}
		}
	}

	// 拿到 admin user_id 用作 updated_by；middleware.JwtAuth 已塞 user_id 到 context
	adminID := int64(middleware.GetUserID(ctx))

	for _, it := range body.Items {
		if err := dao.BotRuntimeConfig().Upsert(
			ctx.Request.Context(), it.Key, it.Value, adminID,
		); err != nil {
			reply.ReplyInternalError(ctx, err)
			return
		}
	}
	reply.ReplyOK(ctx)
}

// 文件最底：避免误用——直接返回 not found 而不是 panic
//
//nolint:unused
func _ifNotFoundReply(ctx *gin.Context, err error) bool {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ctx.AbortWithStatus(http.StatusNotFound)
		return true
	}
	return false
}
