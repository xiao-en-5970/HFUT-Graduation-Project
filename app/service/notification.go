package service

import (
	"context"
	"strings"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/logger"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"go.uber.org/zap"
)

type notificationService struct{}

// Notification 工厂入口：service.Notification().XXX(...)
func Notification() *notificationService { return &notificationService{} }

// emit 的统一入口；内部做了空参容错、自己给自己触发时跳过。
// 所有参数均非严格校验（保持对 emit 调用方的宽容），异常仅写日志不影响主接口。
func (s *notificationService) emit(ctx context.Context, n *model.Notification) {
	if n == nil {
		return
	}
	if n.UserID <= 0 {
		return
	}
	// 自己对自己的行为不通知（例：给自己点赞、给自己评论）
	if n.FromUserID != 0 && n.FromUserID == n.UserID {
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
