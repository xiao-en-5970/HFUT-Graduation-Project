package controller

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// notificationAuthor 发送者缩略信息，嵌入 notificationVO，供前端展示头像与昵称
type notificationAuthor struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

// notificationVO 列表返回的单条通知
type notificationVO struct {
	ID         uint                `json:"id"`
	Type       int                 `json:"type"`
	TargetType int                 `json:"target_type"`
	TargetID   int                 `json:"target_id"`
	RefExtType int                 `json:"ref_ext_type"`
	RefID      int                 `json:"ref_id"`
	Title      string              `json:"title"`
	Summary    string              `json:"summary"`
	Image      string              `json:"image"`
	IsRead     bool                `json:"is_read"`
	CreatedAt  string              `json:"created_at"`
	From       *notificationAuthor `json:"from,omitempty"`
}

// NotificationList GET /notifications?type=all|like|comment|reply|official[,...]&page&page_size&only_unread=1
// type 支持多值逗号分隔；type=all 或缺省表示不过滤
func NotificationList(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	types := parseNotifTypeFilter(ctx.Query("type"))
	onlyUnread := ctx.Query("only_unread") == "1" || ctx.Query("only_unread") == "true"

	list, total, err := dao.Notification().List(ctx.Request.Context(), userID, types, onlyUnread, page, pageSize)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}

	// 填充 from 用户（含 official uid=0）
	fromIDSet := make(map[uint]struct{}, len(list))
	for _, n := range list {
		fromIDSet[uint(n.FromUserID)] = struct{}{}
	}
	fromIDs := make([]uint, 0, len(fromIDSet))
	for id := range fromIDSet {
		fromIDs = append(fromIDs, id)
	}
	userMap, _ := dao.User().GetByIDs(ctx.Request.Context(), fromIDs)

	vos := make([]notificationVO, 0, len(list))
	for _, n := range list {
		v := notificationVO{
			ID:         n.ID,
			Type:       int(n.Type),
			TargetType: int(n.TargetType),
			TargetID:   n.TargetID,
			RefExtType: int(n.RefExtType),
			RefID:      n.RefID,
			Title:      n.Title,
			Summary:    n.Summary,
			Image:      oss.ToFullURL(n.Image),
			IsRead:     n.IsRead,
			CreatedAt:  n.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		// 官方通知：0 号用户可能存在或不存在；都以「官方」展示
		if n.FromUserID == model.OfficialUserID {
			fromVO := &notificationAuthor{ID: 0, Username: "官方", Avatar: ""}
			if u := userMap[0]; u != nil {
				if u.Username != "" {
					fromVO.Username = u.Username
				}
				fromVO.Avatar = u.Avatar
			}
			v.From = fromVO
		} else if u := userMap[uint(n.FromUserID)]; u != nil {
			v.From = &notificationAuthor{
				ID:       u.ID,
				Username: u.Username,
				Avatar:   u.Avatar,
			}
		}
		vos = append(vos, v)
	}

	reply.ReplyOKWithData(ctx, gin.H{
		"list":      vos,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// NotificationUnreadCount GET /notifications/unread_count
// 返回 { total, by_type: {1: n, 2: n, ...} }
func NotificationUnreadCount(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	byType, total, err := dao.Notification().UnreadCountByType(ctx.Request.Context(), userID)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	// 把 type 转成字符串 key，方便前端 JSON 直接用
	m := make(map[string]int64, len(byType))
	for k, v := range byType {
		m[strconv.Itoa(k)] = v
	}
	reply.ReplyOKWithData(ctx, gin.H{
		"total":   total,
		"by_type": m,
	})
}

// NotificationMarkReadReq 标记已读请求体
type NotificationMarkReadReq struct {
	IDs  []uint `json:"ids"`
	All  bool   `json:"all"`  // 全部已读
	Type int    `json:"type"` // 仅 all=true 时生效；0 表示全部类型
}

// NotificationMarkRead POST /notifications/read
func NotificationMarkRead(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	var req NotificationMarkReadReq
	if err := ctx.BindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if req.All {
		if err := dao.Notification().MarkAllRead(ctx.Request.Context(), userID, req.Type); err != nil {
			reply.ReplyInternalError(ctx, err)
			return
		}
		reply.ReplyOK(ctx)
		return
	}
	if len(req.IDs) == 0 {
		reply.ReplyErrWithMessage(ctx, "ids 不能为空，或传 all=true")
		return
	}
	if err := dao.Notification().MarkReadByIDs(ctx.Request.Context(), userID, req.IDs); err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// parseNotifTypeFilter 将 "all"/""/"1,2" 等字符串解析为 type 列表；返回 nil 表示不过滤
func parseNotifTypeFilter(raw string) []int {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "all" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	// alias 便于前端更可读
	alias := map[string]int{
		"like_article": model.NotifyTypeLikeArticle,
		"like_comment": model.NotifyTypeLikeComment,
		"comment":      model.NotifyTypeComment,
		"reply":        model.NotifyTypeReply,
		"official":     model.NotifyTypeOfficial,
		// 聚合：点赞 = like_article + like_comment
		"like": -1,
	}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if v, ok := alias[p]; ok {
			if v == -1 {
				out = append(out, model.NotifyTypeLikeArticle, model.NotifyTypeLikeComment)
			} else {
				out = append(out, v)
			}
			continue
		}
		if n, err := strconv.Atoi(p); err == nil && n >= 1 && n <= 5 {
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
