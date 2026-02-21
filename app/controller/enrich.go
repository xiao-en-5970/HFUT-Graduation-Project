package controller

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
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
