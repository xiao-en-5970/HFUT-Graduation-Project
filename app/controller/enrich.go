package controller

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
)

// enrichArticleWithAuthorForViewer 单篇详情：作者 + 当前用户是否已赞/已藏（extType：1帖 2问 3答）
func enrichArticleWithAuthorForViewer(ctx context.Context, userID uint, extType int, a *model.Article) response.ArticleWithAuthor {
	out := enrichArticleWithAuthor(ctx, a)
	if userID == 0 {
		return out
	}
	aid := int(a.ID)
	if ok, err := dao.Like().Exists(ctx, userID, aid, extType); err == nil {
		out.IsLiked = ok
	}
	if ok, err := dao.CollectItem().ExistsByUserExt(ctx, userID, aid, extType); err == nil {
		out.IsCollected = ok
	}
	return out
}

// enrichArticlesWithAuthor 为文章列表填充作者信息
func enrichArticlesWithAuthor(ctx *gin.Context, list []*model.Article) []response.ArticleWithAuthor {
	if len(list) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(list))
	for _, a := range list {
		if a.UserID != nil && *a.UserID > 0 {
			ids = append(ids, uint(*a.UserID))
		}
	}
	userMap, _ := dao.User().GetByIDsIfValid(ctx.Request.Context(), ids)
	result := make([]response.ArticleWithAuthor, len(list))
	for i, a := range list {
		result[i] = response.ArticleWithAuthor{Article: *a}
		if a.UserID != nil {
			if u := userMap[uint(*a.UserID)]; u != nil {
				result[i].Author = &response.AuthorProfile{
					ID:       u.ID,
					Username: u.Username,
					Avatar:   oss.ToFullURL(u.Avatar),
				}
			}
		}
	}
	return result
}

// enrichArticleWithAuthor 为单篇文章填充作者信息
func enrichArticleWithAuthor(ctx context.Context, a *model.Article) response.ArticleWithAuthor {
	out := response.ArticleWithAuthor{Article: *a}
	if a.UserID != nil && *a.UserID > 0 {
		u, err := dao.User().GetByIDIfValid(ctx, uint(*a.UserID))
		if err == nil && u != nil {
			out.Author = &response.AuthorProfile{
				ID:       u.ID,
				Username: u.Username,
				Avatar:   oss.ToFullURL(u.Avatar),
			}
		}
	}
	return out
}

// enrichCommentsWithAuthor 为评论列表填充作者信息、回复目标作者信息及回复数
func enrichCommentsWithAuthor(ctx *gin.Context, list []*model.Comment) []response.CommentWithAuthor {
	if len(list) == 0 {
		return nil
	}

	userIDs := make([]uint, 0, len(list))
	for _, c := range list {
		if c.UserID != nil && *c.UserID > 0 {
			userIDs = append(userIDs, uint(*c.UserID))
		}
	}

	// 收集回复的目标评论 ID（reply_id 优先，否则 parent_id）
	targetCmtIDSet := make(map[uint]bool)
	for _, c := range list {
		if c.Type == constant.CommentTypeReply {
			if c.ReplyID != nil && *c.ReplyID > 0 {
				targetCmtIDSet[uint(*c.ReplyID)] = true
			} else if c.ParentID != nil && *c.ParentID > 0 {
				targetCmtIDSet[uint(*c.ParentID)] = true
			}
		}
	}
	targetCmtIDs := make([]uint, 0, len(targetCmtIDSet))
	for id := range targetCmtIDSet {
		targetCmtIDs = append(targetCmtIDs, id)
	}

	// 批量获取目标评论，提取其 user_id
	targetCmtMap := make(map[uint]*model.Comment)
	if len(targetCmtIDs) > 0 {
		targets, _ := dao.Comment().GetByIDs(ctx.Request.Context(), targetCmtIDs)
		for _, t := range targets {
			targetCmtMap[t.ID] = t
			if t.UserID != nil && *t.UserID > 0 {
				userIDs = append(userIDs, uint(*t.UserID))
			}
		}
	}

	// 顶层评论批量统计回复数
	var topIDs []uint
	for _, c := range list {
		if c.Type == constant.CommentTypeTop {
			topIDs = append(topIDs, c.ID)
		}
	}
	replyCountMap := make(map[uint]int64)
	if len(topIDs) > 0 {
		replyCountMap, _ = dao.Comment().CountRepliesByParentIDs(ctx.Request.Context(), topIDs)
	}

	userMap, _ := dao.User().GetByIDsIfValid(ctx.Request.Context(), userIDs)

	result := make([]response.CommentWithAuthor, len(list))
	for i, c := range list {
		result[i] = response.CommentWithAuthor{Comment: *c}
		if c.UserID != nil {
			if u := userMap[uint(*c.UserID)]; u != nil {
				result[i].Author = &response.AuthorProfile{
					ID:       u.ID,
					Username: u.Username,
					Avatar:   oss.ToFullURL(u.Avatar),
				}
			}
		}
		if c.Type == constant.CommentTypeTop {
			result[i].ReplyCount = replyCountMap[c.ID]
		}
		if c.Type == constant.CommentTypeReply {
			var tid uint
			if c.ReplyID != nil && *c.ReplyID > 0 {
				tid = uint(*c.ReplyID)
			} else if c.ParentID != nil && *c.ParentID > 0 {
				tid = uint(*c.ParentID)
			}
			if tc := targetCmtMap[tid]; tc != nil && tc.UserID != nil {
				if u := userMap[uint(*tc.UserID)]; u != nil {
					result[i].ReplyToAuthor = &response.AuthorProfile{
						ID:       u.ID,
						Username: u.Username,
						Avatar:   oss.ToFullURL(u.Avatar),
					}
				}
			}
		}
	}
	return result
}

// enrichAnswersWithParentQuestion 回答列表附带所属提问标题与正文（社区流展示用）
func enrichAnswersWithParentQuestion(ctx *gin.Context, viewerSchoolID uint, list []*model.Article) []response.AnswerWithAuthor {
	base := enrichArticlesWithAuthor(ctx, list)
	if len(base) == 0 {
		return nil
	}
	seen := make(map[uint]bool)
	var parentIDs []uint
	for _, a := range list {
		if a.ParentID != nil && *a.ParentID > 0 {
			pid := uint(*a.ParentID)
			if !seen[pid] {
				seen[pid] = true
				parentIDs = append(parentIDs, pid)
			}
		}
	}
	parentMap := make(map[uint]*model.Article)
	for _, pid := range parentIDs {
		q, err := dao.Article().GetByIDWithSchoolOrPublicAndType(ctx.Request.Context(), pid, viewerSchoolID, constant.ArticleTypeQuestion)
		if err == nil && q != nil {
			parentMap[pid] = q
		}
	}
	out := make([]response.AnswerWithAuthor, len(base))
	for i := range base {
		out[i].ArticleWithAuthor = base[i]
		if list[i].ParentID != nil {
			pid := uint(*list[i].ParentID)
			if pq, ok := parentMap[pid]; ok {
				out[i].ParentQuestion = &response.ParentQuestionBrief{
					ID:       pq.ID,
					Title:    pq.Title,
					Content:  pq.Content,
					SchoolID: pq.SchoolID,
				}
			}
		}
	}
	return out
}

// enrichAnswerWithParent 单条回答详情附带提问摘要
func enrichAnswerWithParent(ctx *gin.Context, viewerSchoolID uint, art *model.Article) response.AnswerWithAuthor {
	userID := middleware.GetUserID(ctx)
	base := enrichArticleWithAuthorForViewer(ctx.Request.Context(), userID, constant.ExtTypeAnswer, art)
	out := response.AnswerWithAuthor{ArticleWithAuthor: base}
	if art.ParentID != nil && *art.ParentID > 0 {
		pid := uint(*art.ParentID)
		q, err := dao.Article().GetByIDWithSchoolOrPublicAndType(ctx.Request.Context(), pid, viewerSchoolID, constant.ArticleTypeQuestion)
		if err == nil && q != nil {
			out.ParentQuestion = &response.ParentQuestionBrief{
				ID:       q.ID,
				Title:    q.Title,
				Content:  q.Content,
				SchoolID: q.SchoolID,
			}
		}
	}
	return out
}
