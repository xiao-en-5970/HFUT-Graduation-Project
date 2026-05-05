package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/botinternal"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/logger"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"go.uber.org/zap"
)

type notificationService struct{}

// Notification 工厂入口：service.Notification().XXX(...)
func Notification() *notificationService { return &notificationService{} }

// dispatchInbound 通知的 inbound 分流——按 target 状态决定走"入库" / "孤儿转发到群" / "丢弃"。
//
// 返回 shouldWrite=true 时调用方继续走 DB 入库；false 时已经处理完毕（可能转给 bot
// 或自己给自己 dedupe），直接 return。
//
// 改动 n 的字段（仅在 shouldWrite=true 时生效，调用方据此入库）：
//   - n.UserID  可能从 旗下号 id 改成 parent id（绑定的旗下号情况）
//
// 详见 SKILL.md "数据聚合 / 操作权限" + "孤儿旗下账号特殊行为"段。
func (s *notificationService) dispatchInbound(ctx context.Context, n *model.Notification) (shouldWrite bool) {
	target := ResolveInboundTarget(ctx, uint(n.UserID))
	switch target.Kind {
	case InboundInvalid:
		return false

	case InboundNormal:
		// 普通账号：入库前再做一次 self dedupe
		if n.FromUserID != 0 && n.FromUserID == n.UserID {
			return false
		}
		return true

	case InboundBoundChild:
		// 已绑旗下号：重定向到 parent 后再做一次 self dedupe
		// （场景：主账号 P 点赞了"自己旗下号 C 挂的商品"，重定向后 user_id=P=from_user_id=P，自己通知自己）
		n.UserID = int(target.EffectiveUserID)
		if n.FromUserID != 0 && n.FromUserID == n.UserID {
			return false
		}
		return true

	case InboundOrphan:
		// 孤儿：DB 不入库，bot 转发到原群。失败仅 log，不影响主接口。
		s.forwardOrphanNotice(ctx, n, target)
		return false
	}
	return false
}

// forwardOrphanNotice 孤儿账号收到 inbound 通知时，bot 在原群里 @ 该 QQ 转发"来自 app 用户 X 的 ..."。
//
// 失败处理：
//   - bot internal client 没初始化 → 静默丢弃 + log warn
//   - 没 created_in_group_id（老数据 / 数据异常）→ 静默丢弃 + log info
//   - 没 qq_number → 同上
//   - bot 调用失败 → log warn，不重试（用户在 QQ 没收到也只是错过一次回复，不会数据错乱）
func (s *notificationService) forwardOrphanNotice(ctx context.Context, n *model.Notification, t InboundTarget) {
	if botinternal.Default == nil {
		logger.Infof(ctx, "orphan notification dropped (bot 未配置): user_id=%d type=%d", n.UserID, n.Type)
		return
	}
	if t.OrphanGroupID == 0 || t.OrphanQQNumber == "" {
		logger.Infof(ctx, "orphan notification dropped (无 created_in_group_id 或 qq_number): user_id=%d type=%d", n.UserID, n.Type)
		return
	}

	fromName := s.lookupUsername(ctx, n.FromUserID)
	text := s.composeOrphanText(n, fromName)
	if text == "" {
		return // 类型不支持转发
	}

	var qqInt int64
	fmt.Sscanf(t.OrphanQQNumber, "%d", &qqInt)
	if qqInt == 0 {
		logger.Infof(ctx, "orphan notification dropped (qq_number 解析失败): %q", t.OrphanQQNumber)
		return
	}

	if err := botinternal.Default.SendGroup(ctx, t.OrphanGroupID, qqInt, text); err != nil {
		logger.Warn(ctx, "orphan notification 转发失败",
			zap.Int("user_id", n.UserID), zap.Int("from", n.FromUserID),
			zap.Int16("type", n.Type), zap.Int64("group", t.OrphanGroupID),
			zap.Error(err))
		return
	}
	logger.Infof(ctx, "orphan notification 已转发: user=%d group=%d type=%d", n.UserID, t.OrphanGroupID, n.Type)
}

// lookupUsername 拿 from_user_id 对应的展示名——给孤儿转发文案"来自用户 X" 用。
//
// 找不到时返回 "app 用户"，让转发文案仍能正常发送（不阻断主流程）。
func (s *notificationService) lookupUsername(ctx context.Context, userID int) string {
	if userID <= 0 {
		return "app 用户"
	}
	u, err := dao.User().GetByID(ctx, uint(userID))
	if err != nil || u == nil {
		return "app 用户"
	}
	if u.Username != "" {
		return u.Username
	}
	return "app 用户"
}

// composeOrphanText 按 notification type 拼孤儿转发文案。
//
// 未支持的 type 返回空字符串（调用方不发）；当前覆盖 4 类：
//
//	type=1 点赞文章/商品  → "[bot] 来自用户 X 给你之前发的「TITLE」点了赞"
//	type=2 点赞评论       → "[bot] 来自用户 X 给你的评论点了赞：（评论摘要）"
//	type=3 顶层评论       → "[bot] 来自用户 X 评论了你的「TITLE」：评论内容"
//	type=4 回复评论       → "[bot] 来自用户 X 回复了你的评论：（回复内容）"
//	type=99 官方通知       → 原标题 / 摘要直接发
func (s *notificationService) composeOrphanText(n *model.Notification, fromName string) string {
	title := strings.TrimSpace(n.Title)
	summary := strings.TrimSpace(n.Summary)
	switch n.Type {
	case model.NotifyTypeLikeArticle:
		if title == "" {
			title = "你的内容"
		}
		return fmt.Sprintf("[bot] 来自用户 %s 给你之前发的「%s」点了赞", fromName, title)
	case model.NotifyTypeLikeComment:
		s := summary
		if s == "" {
			s = "(无内容)"
		}
		return fmt.Sprintf("[bot] 来自用户 %s 给你的评论点了赞：%s", fromName, s)
	case model.NotifyTypeComment:
		if title == "" {
			title = "你的内容"
		}
		s := summary
		if s == "" {
			s = "(无评论内容)"
		}
		return fmt.Sprintf("[bot] 来自用户 %s 评论了你的「%s」：%s", fromName, title, s)
	case model.NotifyTypeReply:
		s := summary
		if s == "" {
			s = "(无回复内容)"
		}
		return fmt.Sprintf("[bot] 来自用户 %s 回复了你的评论：%s", fromName, s)
	case model.NotifyTypeOfficial:
		// 官方通知保留原文，加 bot 前缀
		if title != "" && summary != "" {
			return fmt.Sprintf("[bot 官方通知] %s：%s", title, summary)
		}
		if title != "" {
			return fmt.Sprintf("[bot 官方通知] %s", title)
		}
		return fmt.Sprintf("[bot 官方通知] %s", summary)
	}
	return ""
}

// emit 的统一入口；内部做了空参容错、自己给自己触发时跳过。
// 所有参数均非严格校验（保持对 emit 调用方的宽容），异常仅写日志不影响主接口。
//
// **接收人重定向**（QQ 旗下号 → 主账号）：详见 dispatchInbound / SKILL.md "数据聚合 /
// 操作权限"。所有点赞 / 评论 / 回复 / 官方通知都自动适配——单点改造覆盖全链路。
//
// **孤儿转发**：target 是孤儿旗下号时，不写 DB 通知，调 bot 转发到原创建群（详见
// SKILL.md "孤儿旗下账号特殊行为"）。
func (s *notificationService) emit(ctx context.Context, n *model.Notification) {
	if n == nil || n.UserID <= 0 {
		return
	}
	// 早期 self dedupe（在 ResolveInboundTarget 之前先省一次 DB）：相等就直接 return
	if n.FromUserID != 0 && n.FromUserID == n.UserID {
		return
	}
	if !s.dispatchInbound(ctx, n) {
		return
	}
	n.Status = constant.StatusValid
	if err := dao.Notification().Create(ctx, n); err != nil {
		logger.Warn(ctx, "notification emit failed",
			zap.Int("user_id", n.UserID),
			zap.Int("from_user_id", n.FromUserID),
			zap.Int16("type", n.Type),
			zap.Error(err),
		)
	}
}

// emitAggregatedLike 聚合写入点赞通知（type=1/2）。同作品/评论内的多个点赞在"未读窗口"内合并成 1 条。
func (s *notificationService) emitAggregatedLike(ctx context.Context, n *model.Notification) {
	if n == nil || n.UserID <= 0 {
		return
	}
	if n.FromUserID != 0 && n.FromUserID == n.UserID {
		return
	}
	if !s.dispatchInbound(ctx, n) {
		return
	}
	n.Status = constant.StatusValid
	if _, err := dao.Notification().UpsertAggregatedLike(ctx, n); err != nil {
		logger.Warn(ctx, "notification aggregated like emit failed",
			zap.Int("user_id", n.UserID),
			zap.Int("from_user_id", n.FromUserID),
			zap.Int16("type", n.Type),
			zap.Int16("target_type", n.TargetType),
			zap.Int("target_id", n.TargetID),
			zap.Error(err),
		)
	}
}

// EmitLikeArticle 有人点赞了你的帖子/提问/回答/商品
// extType: 1帖子 2提问 3回答 4商品
func (s *notificationService) EmitLikeArticle(ctx context.Context, fromUserID uint, targetExtType int, targetID uint) {
	ownerID, title, summary, image := s.resolveArticleOrGoodOwner(ctx, targetExtType, targetID)
	if ownerID <= 0 {
		return
	}
	go s.emitAggregatedLike(context.Background(), &model.Notification{
		UserID:     ownerID,
		FromUserID: int(fromUserID),
		Type:       model.NotifyTypeLikeArticle,
		TargetType: int16(targetExtType),
		TargetID:   int(targetID),
		Title:      title,
		Summary:    summary,
		Image:      image,
	})
}

// EmitLikeComment 有人点赞了你的评论
func (s *notificationService) EmitLikeComment(ctx context.Context, fromUserID uint, commentID uint) {
	c, err := dao.Comment().GetByID(ctx, commentID)
	if err != nil || c == nil || c.UserID == nil || *c.UserID <= 0 {
		return
	}
	// 评论所属对象，便于前端点击回跳 + 展示「在哪个帖子/回答下」
	_, refTitle, _, _ := s.resolveArticleOrGoodOwner(ctx, c.ExtType, uint(c.ExtID))
	go s.emitAggregatedLike(context.Background(), &model.Notification{
		UserID:     *c.UserID,
		FromUserID: int(fromUserID),
		Type:       model.NotifyTypeLikeComment,
		TargetType: constant.ExtTypeComment,
		TargetID:   int(commentID),
		RefExtType: int16(c.ExtType),
		RefID:      c.ExtID,
		Title:      refTitle,
		Summary:    snippet(c.Content),
	})
}

// EmitComment 有人评论/回复了你的内容
// req 不为 nil 的 ParentID 表示回复；是否回复评论(4) 由 parent 决定：
//   - 顶层评论 → 通知「文章/商品作者」，type=3
//   - 回复评论 → 通知「被回复的评论作者」，type=4
func (s *notificationService) EmitCommentOrReply(
	ctx context.Context,
	fromUserID uint,
	extType int,
	articleID uint,
	commentID uint,
	content string,
	parentID *uint,
	replyID *uint,
) {
	// 回复场景：通知被回复者；被回复者优先级：reply_id > parent_id
	if parentID != nil && *parentID > 0 {
		var targetCommentID uint = *parentID
		if replyID != nil && *replyID > 0 {
			targetCommentID = *replyID
		}
		c, err := dao.Comment().GetByID(ctx, targetCommentID)
		if err != nil || c == nil || c.UserID == nil || *c.UserID <= 0 {
			return
		}
		// 附带回复所在的帖子/回答/商品标题，列表页展示「⟶ 《标题》」
		_, refTitle, _, _ := s.resolveArticleOrGoodOwner(ctx, extType, articleID)
		go s.emit(context.Background(), &model.Notification{
			UserID:     *c.UserID,
			FromUserID: int(fromUserID),
			Type:       model.NotifyTypeReply,
			TargetType: constant.ExtTypeComment,
			TargetID:   int(targetCommentID),
			RefExtType: int16(extType),
			RefID:      int(articleID),
			Title:      refTitle,
			Summary:    snippet(content),
		})
		return
	}

	// 顶层评论：通知作品作者
	ownerID, title, _, image := s.resolveArticleOrGoodOwner(ctx, extType, articleID)
	if ownerID <= 0 {
		return
	}
	go s.emit(context.Background(), &model.Notification{
		UserID:     ownerID,
		FromUserID: int(fromUserID),
		Type:       model.NotifyTypeComment,
		TargetType: int16(extType),
		TargetID:   int(articleID),
		Title:      title,
		Summary:    snippet(content),
		Image:      image,
	})
}

// EmitOfficial 写一条官方通知（FromUserID = 0）；可被后台管理员/脚本直接调用。
func (s *notificationService) EmitOfficial(ctx context.Context, userID uint, title, summary, image string) {
	go s.emit(context.Background(), &model.Notification{
		UserID:     int(userID),
		FromUserID: model.OfficialUserID,
		Type:       model.NotifyTypeOfficial,
		Title:      title,
		Summary:    summary,
		Image:      image,
	})
}

// resolveArticleOrGoodOwner 返回作品作者 ID + 冗余展示字段。
// extType: 1/2/3 文章，4 商品。
func (s *notificationService) resolveArticleOrGoodOwner(ctx context.Context, extType int, id uint) (ownerID int, title, summary, image string) {
	if extType == constant.ExtTypeGoods {
		g, err := dao.Good().GetByID(ctx, id)
		if err != nil || g == nil {
			return 0, "", "", ""
		}
		if g.UserID != nil {
			ownerID = *g.UserID
		}
		title = g.Title
		summary = snippet(g.Content)
		if len(g.Images) > 0 {
			image = g.Images[0]
		}
		return
	}
	a, err := dao.Article().GetByID(ctx, id)
	if err != nil || a == nil {
		return 0, "", "", ""
	}
	if a.UserID != nil {
		ownerID = *a.UserID
	}
	title = a.Title
	summary = snippet(a.Content)
	if len(a.Images) > 0 {
		image = a.Images[0]
	}
	return
}

// snippet 截取最长 120 个 rune 作为摘要，防止超长正文直接写入通知摘要字段。
func snippet(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	const maxRune = 120
	runes := []rune(s)
	if len(runes) <= maxRune {
		return s
	}
	return string(runes[:maxRune]) + "..."
}
