package service

import (
	"errors"
	"mime/multipart"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
)

var (
	ErrArticleNotFoundOrNoPermission = errors.New("帖子不存在或无权限")
	ErrSchoolNotBound                = errors.New("请先绑定学校")
)

type articleService struct{}

func Article() *articleService {
	return &articleService{}
}

// CreateArticleReq 创建帖子请求
type CreateArticleReq struct {
	Title         string `json:"title" binding:"required"`
	Content       string `json:"content" binding:"required"`
	Type          int    `json:"type"`           // 1:普通 2:提问 3:回答
	PublishStatus int16  `json:"publish_status"` // 1:私密 2:公开
	SchoolID      *int   `json:"school_id"`
}

// UpdateArticleReq 更新帖子请求
type UpdateArticleReq struct {
	Title         *string `json:"title"`
	Content       *string `json:"content"`
	Type          *int    `json:"type"`
	PublishStatus *int16  `json:"publish_status"`
	SchoolID      *int    `json:"school_id"`
}

func (s *articleService) Create(ctx *gin.Context, userID uint, schoolID uint, req CreateArticleReq) (uint, error) {
	if schoolID == 0 {
		return 0, ErrSchoolNotBound
	}
	uid := int(userID)
	sid := int(schoolID)
	a := &model.Article{
		UserID:        &uid,
		SchoolID:      &sid, // 强制使用用户学校，不得跨校
		Title:         req.Title,
		Content:       req.Content,
		Status:        constant.StatusValid,
		PublishStatus: req.PublishStatus,
		Type:          req.Type,
	}
	if a.PublishStatus == 0 {
		a.PublishStatus = 1
	}
	if a.Type == 0 {
		a.Type = constant.ArticleTypeNormal
	}
	return dao.Article().Create(ctx.Request.Context(), a)
}

func (s *articleService) Get(ctx *gin.Context, id uint, viewerID uint, schoolID uint) (*model.Article, error) {
	if schoolID == 0 {
		return nil, ErrSchoolNotBound
	}
	// 学校隔离：仅能查看本校帖子
	art, err := dao.Article().GetByIDWithSchool(ctx.Request.Context(), id, schoolID)
	if err != nil {
		return nil, err
	}
	// 私密帖子仅作者可见
	if art.PublishStatus == 1 {
		ok, _ := dao.Article().ExistsAndOwnedByWithSchool(ctx.Request.Context(), id, viewerID, schoolID)
		if !ok {
			return nil, ErrArticleNotFoundOrNoPermission
		}
	}
	return art, nil
}

func (s *articleService) Update(ctx *gin.Context, id uint, userID uint, schoolID uint, req UpdateArticleReq) error {
	ok, err := dao.Article().ExistsAndOwnedByWithSchool(ctx.Request.Context(), id, userID, schoolID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrArticleNotFoundOrNoPermission
	}
	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.Type != nil {
		updates["type"] = *req.Type
	}
	if req.PublishStatus != nil {
		updates["publish_status"] = *req.PublishStatus
	}
	// 禁止跨校修改 school_id
	if req.SchoolID != nil && schoolID > 0 && uint(*req.SchoolID) == schoolID {
		updates["school_id"] = *req.SchoolID
	}
	if len(updates) == 0 {
		return nil
	}
	return dao.Article().UpdateColumns(ctx.Request.Context(), id, updates)
}

// UploadImages 批量上传帖子图片，按顺序保存为 image_1, image_2, ...
func (s *articleService) UploadImages(ctx *gin.Context, id uint, userID uint, schoolID uint, files []*multipart.FileHeader) ([]string, error) {
	ok, err := dao.Article().ExistsAndOwnedByWithSchool(ctx.Request.Context(), id, userID, schoolID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrArticleNotFoundOrNoPermission
	}
	if len(files) == 0 {
		return nil, errors.New("至少需要上传一张图片")
	}
	urls := make([]string, 0, len(files))
	for i, f := range files {
		ext := oss.ExtFromFilename(f.Filename)
		relPath := oss.ArticleImagePath(id, i+1, ext)
		url, err := oss.Save(f, relPath)
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	if err := dao.Article().UpdateImages(ctx.Request.Context(), id, urls); err != nil {
		return nil, err
	}
	return urls, nil
}

func (s *articleService) UpdateImages(ctx *gin.Context, id uint, userID uint, schoolID uint, images []string) error {
	ok, err := dao.Article().ExistsAndOwnedByWithSchool(ctx.Request.Context(), id, userID, schoolID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrArticleNotFoundOrNoPermission
	}
	return dao.Article().UpdateImages(ctx.Request.Context(), id, images)
}

func (s *articleService) Delete(ctx *gin.Context, id uint, userID uint, schoolID uint) error {
	ok, err := dao.Article().ExistsAndOwnedByWithSchool(ctx.Request.Context(), id, userID, schoolID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrArticleNotFoundOrNoPermission
	}
	return dao.Article().SoftDelete(ctx.Request.Context(), id)
}

func (s *articleService) List(ctx *gin.Context, schoolID uint, page, pageSize int) ([]*model.Article, int64, error) {
	if schoolID == 0 {
		return nil, 0, ErrSchoolNotBound
	}
	return dao.Article().List(ctx.Request.Context(), schoolID, page, pageSize)
}

func (s *articleService) Search(ctx *gin.Context, schoolID uint, keyword string, page, pageSize int) ([]*model.Article, int64, error) {
	if schoolID == 0 {
		return nil, 0, ErrSchoolNotBound
	}
	return dao.Article().Search(ctx.Request.Context(), schoolID, keyword, page, pageSize)
}
