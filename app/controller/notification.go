package controller

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// notifAccountIDs caller 在做"我的通知"操作时能 access 的 user_id 集合（含旗下号）。
//
// 失败时降级到 [callerID]——通知功能不该因为聚合查询出错而整体挂掉。
func notifAccountIDs(ctx *gin.Context, callerID uint) []uint {
	ids, err := service.GetAccountIDsForOps(ctx.Request.Context(), callerID)
	if err != nil || !ids.IsAggregated() {
		return []uint{callerID}
	}
	return ids.AllIDs
}

// notificationAuthor 发送者缩略信息，嵌入 notificationVO，供前端展示头像与昵称。
//
// QQ 旗下号语义：跟 vo.AuthorProfile 一致——非孤儿旗下号作为通知发起者时，会额外
// 带 from_user_id + from_username 让前端拼"username（来自用户 xxx）"展示。
// 这样"我点赞 / 评论 / 回复 / 关注的人是 X 通过 QQ 发的"在通知里也可见。
type notificationAuthor struct {
	ID           uint   `json:"id"`
	Username     string `json:"username"`
	Avatar       string `json:"avatar"`
	FromUserID   uint   `json:"from_user_id,omitempty"`
	FromUsername string `json:"from_username,omitempty"`
}

// notificationVO 列表返回的单条通知
type notificationVO struct {
	ID         uint   `json:"id"`
	Type       int    `json:"type"`
	TargetType int    `json:"target_type"`
	TargetID   int    `json:"target_id"`
	RefExtType int    `json:"ref_ext_type"`
	RefID      int    `json:"ref_id"`
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	Image      string `json:"image"`
	IsRead     bool   `json:"is_read"`
	// Count 聚合触发者人数：点赞类可能 >1（N 人点赞了你）；顶层评论与回复始终 =1。
	Count     int                 `json:"count"`
	CreatedAt string              `json:"created_at"`
	UpdatedAt string              `json:"updated_at"`
	From      *notificationAuthor `json:"from,omitempty"`
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

	list, total, err := dao.Notification().ListByUserIDs(ctx.Request.Context(), notifAccountIDs(ctx, userID), types, onlyUnread, page, pageSize)
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
	// 非孤儿 QQ 旗下号 → 拉它们的主账号信息，用来拼"（来自用户 xxx）"——跟
	// buildAuthorVO 走同一套语义；详见 vo/response.AuthorProfile 注释。
	parentMap := make(map[uint]*model.User, 0)
	if len(userMap) > 0 {
		validMap := make(map[uint]*model.User, len(userMap))
		for id, u := range userMap {
			if u != nil {
				validMap[id] = u
			}
		}
		parentMap = fetchAuthorParents(ctx.Request.Context(), validMap)
	}

	vos := make([]notificationVO, 0, len(list))
	for _, n := range list {
		cnt := n.Count
		if cnt <= 0 {
			cnt = 1
		}
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
			Count:      cnt,
			CreatedAt:  n.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:  n.UpdatedAt.Format("2006-01-02 15:04:05"),
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
			fromVO := &notificationAuthor{
				ID:       u.ID,
				Username: u.Username,
				Avatar:   u.Avatar,
			}
			if u.IsQQChild() && u.ParentUserID != nil && *u.ParentUserID > 0 {
				if parent := parentMap[uint(*u.ParentUserID)]; parent != nil {
					fromVO.FromUserID = parent.ID
					fromVO.FromUsername = parent.Username
				}
			}
			v.From = fromVO
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
	byType, total, err := dao.Notification().UnreadCountByTypeForUsers(ctx.Request.Context(), notifAccountIDs(ctx, userID))
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
		if err := dao.Notification().MarkAllReadForUsers(ctx.Request.Context(), notifAccountIDs(ctx, userID), req.Type); err != nil {
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
	if err := dao.Notification().MarkReadByIDsForUsers(ctx.Request.Context(), notifAccountIDs(ctx, userID), req.IDs); err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// parseNotifTypeFilter 将 "all"/""/"1,2" 等字符串解析为 type 列表；返回 nil 表示不过滤
//
// 聚合 alias 约定（前端直接用 "comment"/"like"/"official" 这类短名，可读性更好）：
//   - "like"    = like_article + like_comment
//   - "comment" = comment + reply（回复归入评论分类，UI 只保留「评论」一个 tab）
func parseNotifTypeFilter(raw string) []int {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "all" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	// 聚合 alias 的值用负数占位，见下方展开逻辑
	const aliasLike = -1
	const aliasComment = -2
	alias := map[string]int{
		"like_article": model.NotifyTypeLikeArticle,
		"like_comment": model.NotifyTypeLikeComment,
		"official":     model.NotifyTypeOfficial,
		// 保留 reply/comment_only 精确筛选能力，后台管理或调试时会用
		"comment_only": model.NotifyTypeComment,
		"reply":        model.NotifyTypeReply,
		// 聚合类型
		"like":    aliasLike,
		"comment": aliasComment,
	}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if v, ok := alias[p]; ok {
			switch v {
			case aliasLike:
				out = append(out, model.NotifyTypeLikeArticle, model.NotifyTypeLikeComment)
			case aliasComment:
				out = append(out, model.NotifyTypeComment, model.NotifyTypeReply)
			default:
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
