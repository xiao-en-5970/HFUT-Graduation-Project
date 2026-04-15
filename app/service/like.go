package service

import (
	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service/errno"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"gorm.io/gorm"
)

type likeService struct{}

// AddArticle 点赞
// extType: 1帖子 2提问 3回答 4商品
func (s *likeService) AddArticle(ctx *gin.Context, userID uint, schoolID uint, articleID uint, extType int) error {
	if extType == constant.ExtTypeGoods {
		return s.addGoodLike(ctx, userID, schoolID, articleID)
	}
	art, err := dao.Article().GetByIDWithSchoolAndType(ctx.Request.Context(), articleID, schoolID, extType)
	if err != nil || art == nil {
		return errno.ErrLikeArticleNotFound
	}
	if art.PublishStatus == 1 {
		ok, _ := dao.Article().ExistsAndOwnedByWithSchoolAndType(ctx.Request.Context(), articleID, userID, schoolID, extType)
		if !ok {
			return errno.ErrLikeArticleNotFound
		}
	}
	// 惰性新建：若存在 status=2 的记录则恢复，否则新建；已点赞则幂等返回成功
	exist, getErr := dao.Like().GetByUserExt(ctx.Request.Context(), userID, int(articleID), extType)
	if getErr == nil {
		if exist.Status == constant.StatusValid {
			return nil // 幂等：已点赞，多次点赞视为成功
		}
		// status=2，恢复并更新计数
		return pgsql.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
			if err := dao.Like().RestoreWithDB(tx, userID, int(articleID), extType); err != nil {
				return err
			}
			return dao.Article().UpdateLikeCountDB(tx, articleID, 1)
		})
	}
	if getErr != gorm.ErrRecordNotFound {
		return getErr
	}
	// 记录不存在，新建（唯一约束防止并发重复，若冲突则幂等返回）
	uid := int(userID)
	l := &model.Like{
		UserID:  &uid,
		ExtID:   int(articleID),
		ExtType: extType,
		Status:  constant.StatusValid,
	}
	err = pgsql.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := dao.Like().CreateWithDB(tx, l); err != nil {
			return err
		}
		return dao.Article().UpdateLikeCountDB(tx, articleID, 1)
	})
	if err != nil {
		// 并发时可能触发唯一约束冲突，此时记录已存在，视为幂等成功
		exist2, _ := dao.Like().GetByUserExt(ctx.Request.Context(), userID, int(articleID), extType)
		if exist2 != nil && exist2.Status == constant.StatusValid {
			return nil
		}
		return err
	}
	return nil
}

// addGoodLike 点赞商品
func (s *likeService) addGoodLike(ctx *gin.Context, userID uint, schoolID uint, goodID uint) error {
	g, err := dao.Good().GetByIDWithSchool(ctx.Request.Context(), goodID, schoolID)
	if err != nil || g == nil {
		return errno.ErrLikeArticleNotFound
	}
	_ = g
	exist, getErr := dao.Like().GetByUserExt(ctx.Request.Context(), userID, int(goodID), constant.ExtTypeGoods)
	if getErr == nil {
		if exist.Status == constant.StatusValid {
			return nil
		}
		return pgsql.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
			if err := dao.Like().RestoreWithDB(tx, userID, int(goodID), constant.ExtTypeGoods); err != nil {
				return err
			}
			return dao.Good().UpdateLikeCountDB(tx, goodID, 1)
		})
	}
	if getErr != gorm.ErrRecordNotFound {
		return getErr
	}
	uid := int(userID)
	l := &model.Like{
		UserID:  &uid,
		ExtID:   int(goodID),
		ExtType: constant.ExtTypeGoods,
		Status:  constant.StatusValid,
	}
	err = pgsql.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := dao.Like().CreateWithDB(tx, l); err != nil {
			return err
		}
		return dao.Good().UpdateLikeCountDB(tx, goodID, 1)
	})
	if err != nil {
		exist2, _ := dao.Like().GetByUserExt(ctx.Request.Context(), userID, int(goodID), constant.ExtTypeGoods)
		if exist2 != nil && exist2.Status == constant.StatusValid {
			return nil
		}
		return err
	}
	return nil
}

// AddComment 评论点赞
func (s *likeService) AddComment(ctx *gin.Context, userID uint, commentID uint) error {
	c, err := dao.Comment().GetByID(ctx.Request.Context(), commentID)
	if err != nil || c == nil {
		return errno.ErrLikeArticleNotFound
	}
	exist, getErr := dao.Like().GetByUserExt(ctx.Request.Context(), userID, int(commentID), constant.ExtTypeComment)
	if getErr == nil {
		if exist.Status == constant.StatusValid {
			return nil
		}
		return pgsql.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
			if err := dao.Like().RestoreWithDB(tx, userID, int(commentID), constant.ExtTypeComment); err != nil {
				return err
			}
			return dao.Comment().UpdateLikeCountDB(tx, commentID, 1)
		})
	}
	if getErr != gorm.ErrRecordNotFound {
		return getErr
	}
	uid := int(userID)
	l := &model.Like{
		UserID:  &uid,
		ExtID:   int(commentID),
		ExtType: constant.ExtTypeComment,
		Status:  constant.StatusValid,
	}
	err = pgsql.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := dao.Like().CreateWithDB(tx, l); err != nil {
			return err
		}
		return dao.Comment().UpdateLikeCountDB(tx, commentID, 1)
	})
	if err != nil {
		exist2, _ := dao.Like().GetByUserExt(ctx.Request.Context(), userID, int(commentID), constant.ExtTypeComment)
		if exist2 != nil && exist2.Status == constant.StatusValid {
			return nil
		}
		return err
	}
	return nil
}

// RemoveComment 取消评论点赞
func (s *likeService) RemoveComment(ctx *gin.Context, userID uint, commentID uint) error {
	c, err := dao.Comment().GetByID(ctx.Request.Context(), commentID)
	if err != nil || c == nil {
		return errno.ErrLikeArticleNotFound
	}
	ok, _ := dao.Like().Exists(ctx.Request.Context(), userID, int(commentID), constant.ExtTypeComment)
	if !ok {
		return nil
	}
	return pgsql.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := dao.Like().SoftDeleteWithDB(tx, userID, int(commentID), constant.ExtTypeComment); err != nil {
			return err
		}
		return dao.Comment().UpdateLikeCountDB(tx, commentID, -1)
	})
}

// RemoveArticle 取消点赞
func (s *likeService) RemoveArticle(ctx *gin.Context, userID uint, schoolID uint, articleID uint, extType int) error {
	if extType == constant.ExtTypeGoods {
		_, err := dao.Good().GetByIDWithSchool(ctx.Request.Context(), articleID, schoolID)
		if err != nil {
			return errno.ErrLikeArticleNotFound
		}
		ok, _ := dao.Like().Exists(ctx.Request.Context(), userID, int(articleID), extType)
		if !ok {
			return nil
		}
		return pgsql.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
			if err := dao.Like().SoftDeleteWithDB(tx, userID, int(articleID), extType); err != nil {
				return err
			}
			return dao.Good().UpdateLikeCountDB(tx, articleID, -1)
		})
	}
	_, err := dao.Article().GetByIDWithSchoolAndType(ctx.Request.Context(), articleID, schoolID, extType)
	if err != nil {
		return errno.ErrLikeArticleNotFound
	}
	ok, _ := dao.Like().Exists(ctx.Request.Context(), userID, int(articleID), extType)
	if !ok {
		return nil
	}
	return pgsql.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := dao.Like().SoftDeleteWithDB(tx, userID, int(articleID), extType); err != nil {
			return err
		}
		return dao.Article().UpdateLikeCountDB(tx, articleID, -1)
	})
}
