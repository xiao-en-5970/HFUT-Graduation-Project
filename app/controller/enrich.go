package controller

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
)

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

// enrichCommentsWithAuthor 为评论列表填充作者信息
func enrichCommentsWithAuthor(ctx *gin.Context, list []*model.Comment) []response.CommentWithAuthor {
	if len(list) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(list))
	for _, c := range list {
		if c.UserID != nil && *c.UserID > 0 {
			ids = append(ids, uint(*c.UserID))
		}
	}
	userMap, _ := dao.User().GetByIDsIfValid(ctx.Request.Context(), ids)
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
	base := enrichArticleWithAuthor(ctx.Request.Context(), art)
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
