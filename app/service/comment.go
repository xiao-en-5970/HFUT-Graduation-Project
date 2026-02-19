package service

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"gorm.io/gorm"
)

var (
	ErrCommentArticleNotFound = errors.New("文章不存在或无权限")
	ErrCommentParentNotFound  = errors.New("父评论不存在")
)

type commentService struct{}

func Comment() *commentService {
	return &commentService{}
}

// CreateCommentReq 发评论/回复请求
// 不传 parent_id 为顶层评论；传 parent_id 为回复某评论，可选 reply_id 表示回复某条回复
type CreateCommentReq struct {
	Content  string `json:"content" binding:"required"`
	ParentID *uint  `json:"parent_id"` // 顶层评论不传；回复时传父评论 ID
	ReplyID  *uint  `json:"reply_id"`  // 回复某条回复时传被回复的评论 ID
}

// Create 发评论或回复
// articleID 为文章 ID，extType 为 1帖子/2提问/3回答，需校验文章存在且同校
func (s *commentService) Create(ctx *gin.Context, userID uint, schoolID uint, articleID uint, extType int, req CreateCommentReq) (uint, error) {
	if schoolID == 0 {
		return 0, ErrSchoolNotBound
	}
	// 校验文章存在、同校且类型匹配
	art, err := dao.Article().GetByIDWithSchoolAndType(ctx.Request.Context(), articleID, schoolID, extType)
	if err != nil || art == nil {
		if err == gorm.ErrRecordNotFound {
			return 0, ErrCommentArticleNotFound
		}
		return 0, err
	}
	// 文章需为公开或用户自己的
	if art.PublishStatus == 1 {
		ok, _ := dao.Article().ExistsAndOwnedByWithSchoolAndType(ctx.Request.Context(), articleID, userID, schoolID, extType)
		if !ok {
			return 0, ErrCommentArticleNotFound
		}
	}

	uid := int(userID)
	aid := int(articleID)
	c := &model.Comment{
		UserID:  &uid,
		ExtType: extType,
		ExtID:   aid,
		Content: req.Content,
		Status:  constant.StatusValid,
		Type:    constant.CommentTypeTop,
	}

	if req.ParentID != nil && *req.ParentID > 0 {
		// 回复：parent_id 为顶层评论 ID
		parent, err := dao.Comment().GetByID(ctx.Request.Context(), *req.ParentID)
		if err != nil || parent == nil {
			return 0, ErrCommentParentNotFound
		}
		if parent.ExtType != extType || parent.ExtID != aid {
			return 0, ErrCommentParentNotFound
		}
		if parent.Type != constant.CommentTypeTop {
			return 0, ErrCommentParentNotFound
		}
		pid := int(*req.ParentID)
		c.ParentID = &pid
		c.Type = constant.CommentTypeReply
		if req.ReplyID != nil && *req.ReplyID > 0 && *req.ReplyID != *req.ParentID {
			// 回复某条回复：校验 reply 属于该 parent
			replyComment, err := dao.Comment().GetByID(ctx.Request.Context(), *req.ReplyID)
			if err != nil || replyComment == nil {
				return 0, ErrCommentParentNotFound
			}
			if replyComment.ParentID == nil || uint(*replyComment.ParentID) != *req.ParentID {
				return 0, ErrCommentParentNotFound
			}
			rid := int(*req.ReplyID)
			c.ReplyID = &rid
		}
	}

	return dao.Comment().Create(ctx.Request.Context(), c)
}

// ListComments 某文章的顶层评论列表
// extType 1帖子 2提问 3回答
func (s *commentService) ListComments(ctx *gin.Context, userID uint, schoolID uint, articleID uint, extType int, page, pageSize int) ([]*model.Comment, int64, error) {
	if schoolID == 0 {
		return nil, 0, ErrSchoolNotBound
	}
	art, err := dao.Article().GetByIDWithSchoolAndType(ctx.Request.Context(), articleID, schoolID, extType)
	if err != nil || art == nil {
		return nil, 0, ErrCommentArticleNotFound
	}
	if art.PublishStatus == 1 {
		ok, _ := dao.Article().ExistsAndOwnedByWithSchoolAndType(ctx.Request.Context(), articleID, userID, schoolID, extType)
		if !ok {
			return nil, 0, ErrCommentArticleNotFound
		}
	}
	return dao.Comment().ListTopByExt(ctx.Request.Context(), extType, int(articleID), page, pageSize)
}

// ListReplies 某评论的回复列表
// extType 1帖子 2提问 3回答
func (s *commentService) ListReplies(ctx *gin.Context, userID uint, schoolID uint, articleID uint, commentID uint, extType int, page, pageSize int) ([]*model.Comment, int64, error) {
	if schoolID == 0 {
		return nil, 0, ErrSchoolNotBound
	}
	art, err := dao.Article().GetByIDWithSchoolAndType(ctx.Request.Context(), articleID, schoolID, extType)
	if err != nil || art == nil {
		return nil, 0, ErrCommentArticleNotFound
	}
	if art.PublishStatus == 1 {
		ok, _ := dao.Article().ExistsAndOwnedByWithSchoolAndType(ctx.Request.Context(), articleID, userID, schoolID, extType)
		if !ok {
			return nil, 0, ErrCommentArticleNotFound
		}
	}
	ok, err := dao.Comment().ExistsByExtAndID(ctx.Request.Context(), extType, int(articleID), commentID)
	if err != nil || !ok {
		return nil, 0, ErrCommentParentNotFound
	}
	return dao.Comment().ListRepliesByParentID(ctx.Request.Context(), commentID, page, pageSize)
}
