package service

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
)

var (
	ErrLikeArticleNotFound = errors.New("文章不存在")
	ErrLikeAlreadyLiked    = errors.New("已点赞")
	ErrLikeNotLiked        = errors.New("未点赞")
)

type likeService struct{}

func Like() *likeService {
	return &likeService{}
}

// AddArticle 点赞文章
// extType: 1帖子 2提问 3回答
func (s *likeService) AddArticle(ctx *gin.Context, userID uint, schoolID uint, articleID uint, extType int) error {
	art, err := dao.Article().GetByIDWithSchoolAndType(ctx.Request.Context(), articleID, schoolID, extType)
	if err != nil || art == nil {
		return ErrLikeArticleNotFound
	}
	if art.PublishStatus == 1 {
		ok, _ := dao.Article().ExistsAndOwnedByWithSchoolAndType(ctx.Request.Context(), articleID, userID, schoolID, extType)
		if !ok {
			return ErrLikeArticleNotFound
		}
	}
	ok, _ := dao.Like().Exists(ctx.Request.Context(), userID, int(articleID), extType)
	if ok {
		return ErrLikeAlreadyLiked
	}
	uid := int(userID)
	l := &model.Like{
		UserID:  &uid,
		ExtID:   int(articleID),
		ExtType: extType,
		Status:  constant.StatusValid,
	}
	_, err = dao.Like().Create(ctx.Request.Context(), l)
	if err != nil {
		return err
	}
	return dao.Article().UpdateLikeCount(ctx.Request.Context(), articleID, 1)
}

// RemoveArticle 取消点赞
func (s *likeService) RemoveArticle(ctx *gin.Context, userID uint, schoolID uint, articleID uint, extType int) error {
	_, err := dao.Article().GetByIDWithSchoolAndType(ctx.Request.Context(), articleID, schoolID, extType)
	if err != nil {
		return ErrLikeArticleNotFound
	}
	ok, _ := dao.Like().Exists(ctx.Request.Context(), userID, int(articleID), extType)
	if !ok {
		return ErrLikeNotLiked
	}
	err = dao.Like().Delete(ctx.Request.Context(), userID, int(articleID), extType)
	if err != nil {
		return err
	}
	return dao.Article().UpdateLikeCount(ctx.Request.Context(), articleID, -1)
}
