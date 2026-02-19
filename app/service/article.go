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
	ErrParentQuestionRequired        = errors.New("回答必须指定父提问 parent_id")
	ErrParentQuestionNotFound        = errors.New("父提问不存在或非本校")
)

type articleService struct{}

// CreateArticleReq 创建请求（type 由接口路径决定）
// 回答类型必须传 parent_id 指向提问
type CreateArticleReq struct {
	Title         string `json:"title" binding:"required"`
	Content       string `json:"content" binding:"required"`
	PublishStatus int16  `json:"publish_status"` // 1:私密 2:公开
	ParentID      *uint  `json:"parent_id"`      // 仅回答必传，指向提问ID
}

// UpdateArticleReq 更新请求（type 不可修改，由接口隔离）
type UpdateArticleReq struct {
	Title         *string `json:"title"`
	Content       *string `json:"content"`
	PublishStatus *int16  `json:"publish_status"`
}

// Create 创建，articleType 由调用方传入（1帖子 2提问 3回答），学校强制隔离
// 回答必须指定 parent_id 且父文章为提问、同校
func (s *articleService) Create(ctx *gin.Context, userID uint, schoolID uint, articleType int, req CreateArticleReq) (uint, error) {
	if schoolID == 0 {
		return 0, ErrSchoolNotBound
	}
	if articleType == constant.ArticleTypeAnswer {
		if req.ParentID == nil || *req.ParentID == 0 {
			return 0, ErrParentQuestionRequired
		}
		parent, err := dao.Article().GetByIDWithSchoolAndType(ctx.Request.Context(), *req.ParentID, schoolID, constant.ArticleTypeQuestion)
		if err != nil || parent == nil {
			return 0, ErrParentQuestionNotFound
		}
	}
	uid := int(userID)
	sid := int(schoolID)
	a := &model.Article{
		UserID:        &uid,
		SchoolID:      &sid,
		Title:         req.Title,
		Content:       req.Content,
		Status:        constant.StatusValid,
		PublishStatus: req.PublishStatus,
		Type:          articleType,
	}
	if a.PublishStatus == 0 {
		a.PublishStatus = 1
	}
	if articleType == constant.ArticleTypeAnswer && req.ParentID != nil {
		pid := int(*req.ParentID)
		a.ParentID = &pid
	}
	return dao.Article().Create(ctx.Request.Context(), a)
}

// Get 获取详情，学校+类型隔离
func (s *articleService) Get(ctx *gin.Context, id uint, viewerID uint, schoolID uint, articleType int) (*model.Article, error) {
	if schoolID == 0 {
		return nil, ErrSchoolNotBound
	}
	art, err := dao.Article().GetByIDWithSchoolAndType(ctx.Request.Context(), id, schoolID, articleType)
	if err != nil {
		return nil, err
	}
	if art.PublishStatus == 1 {
		ok, _ := dao.Article().ExistsAndOwnedByWithSchoolAndType(ctx.Request.Context(), id, viewerID, schoolID, articleType)
		if !ok {
			return nil, ErrArticleNotFoundOrNoPermission
		}
	}
	return art, nil
}

// Update 更新，类型不可修改
func (s *articleService) Update(ctx *gin.Context, id uint, userID uint, schoolID uint, articleType int, req UpdateArticleReq) error {
	ok, err := dao.Article().ExistsAndOwnedByWithSchoolAndType(ctx.Request.Context(), id, userID, schoolID, articleType)
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
	if req.PublishStatus != nil {
		updates["publish_status"] = *req.PublishStatus
	}
	if len(updates) == 0 {
		return nil
	}
	return dao.Article().UpdateColumns(ctx.Request.Context(), id, updates)
}

// UploadImages 批量上传图片，类型隔离
func (s *articleService) UploadImages(ctx *gin.Context, id uint, userID uint, schoolID uint, articleType int, files []*multipart.FileHeader) ([]string, error) {
	ok, err := dao.Article().ExistsAndOwnedByWithSchoolAndType(ctx.Request.Context(), id, userID, schoolID, articleType)
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
	paths := make([]string, len(urls))
	for i, u := range urls {
		paths[i] = oss.PathForStorage(u)
	}
	if err := dao.Article().UpdateImages(ctx.Request.Context(), id, paths); err != nil {
		return nil, err
	}
	return urls, nil
}

// UpdateImages 更新图片URL元数据，类型隔离
func (s *articleService) UpdateImages(ctx *gin.Context, id uint, userID uint, schoolID uint, articleType int, images []string) error {
	ok, err := dao.Article().ExistsAndOwnedByWithSchoolAndType(ctx.Request.Context(), id, userID, schoolID, articleType)
	if err != nil {
		return err
	}
	if !ok {
		return ErrArticleNotFoundOrNoPermission
	}
	paths := make([]string, len(images))
	for i, p := range images {
		paths[i] = oss.PathForStorage(p)
	}
	return dao.Article().UpdateImages(ctx.Request.Context(), id, paths)
}

// Delete 软删除，类型隔离
func (s *articleService) Delete(ctx *gin.Context, id uint, userID uint, schoolID uint, articleType int) error {
	ok, err := dao.Article().ExistsAndOwnedByWithSchoolAndType(ctx.Request.Context(), id, userID, schoolID, articleType)
	if err != nil {
		return err
	}
	if !ok {
		return ErrArticleNotFoundOrNoPermission
	}
	return dao.Article().SoftDelete(ctx.Request.Context(), id)
}

// List 分页列表，学校+类型隔离
func (s *articleService) List(ctx *gin.Context, schoolID uint, articleType int, page, pageSize int) ([]*model.Article, int64, error) {
	if schoolID == 0 {
		return nil, 0, ErrSchoolNotBound
	}
	return dao.Article().List(ctx.Request.Context(), schoolID, articleType, page, pageSize)
}

// Search 全文检索，学校+类型隔离
func (s *articleService) Search(ctx *gin.Context, schoolID uint, articleType int, keyword string, page, pageSize int) ([]*model.Article, int64, error) {
	if schoolID == 0 {
		return nil, 0, ErrSchoolNotBound
	}
	return dao.Article().Search(ctx.Request.Context(), schoolID, articleType, keyword, page, pageSize)
}

// ListAnswersByQuestionID 列出某提问下的回答，学校隔离
func (s *articleService) ListAnswersByQuestionID(ctx *gin.Context, questionID uint, schoolID uint, page, pageSize int) ([]*model.Article, int64, error) {
	if schoolID == 0 {
		return nil, 0, ErrSchoolNotBound
	}
	// 校验提问存在且为提问类型
	_, err := dao.Article().GetByIDWithSchoolAndType(ctx.Request.Context(), questionID, schoolID, constant.ArticleTypeQuestion)
	if err != nil {
		return nil, 0, ErrParentQuestionNotFound
	}
	return dao.Article().ListByParentID(ctx.Request.Context(), questionID, schoolID, constant.ArticleTypeAnswer, page, pageSize)
}
