package service

import (
	"errors"
	"mime/multipart"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/snowflake"
)

var (
	ErrArticleNotFoundOrNoPermission = errors.New("帖子不存在或无权限")
	ErrSchoolNotBound                = errors.New("请先绑定学校")
	ErrParentQuestionRequired        = errors.New("回答必须指定父提问 parent_id")
	ErrParentQuestionNotFound        = errors.New("父提问不存在或非本校")
	ErrDraftNotFoundOrNoPermission   = errors.New("草稿不存在或无权限")
)

type articleService struct{}

// CreateArticleReq 创建请求（type 由接口路径决定）
// 创建草稿：仅需元信息，返回 ID 供后续编辑和 OSS 上传。回答类型必须传 parent_id
type CreateArticleReq struct {
	Title         string `json:"title"`          // 可选，草稿可空
	Content       string `json:"content"`        // 可选，草稿可空
	PublishStatus int16  `json:"publish_status"` // 1:私密 2:公开
	ParentID      *uint  `json:"parent_id"`      // 仅回答必传，指向提问ID
}

// UpdateArticleReq 更新请求（type 不可修改，由接口隔离）
type UpdateArticleReq struct {
	Title         *string   `json:"title"`
	Content       *string   `json:"content"`
	PublishStatus *int16    `json:"publish_status"`
	Status        *int16    `json:"status"` // 1:发布 3:草稿，草稿可改为发布
	Images        *[]string `json:"images"` // 图片 URL 列表，按顺序覆盖
}

// Create 创建草稿，articleType 由调用方传入。仅存 user_id、school_id、type、parent_id（回答）等元信息
// 返回 ID 供前端立即进行 OSS 上传和编辑。回答必须指定 parent_id
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
		Status:        constant.StatusDraft,
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

// Get 获取详情，学校+类型隔离。草稿仅作者可见
func (s *articleService) Get(ctx *gin.Context, id uint, viewerID uint, schoolID uint, articleType int) (*model.Article, error) {
	if schoolID == 0 {
		return nil, ErrSchoolNotBound
	}
	art, err := dao.Article().GetByIDWithSchoolAndTypeAllowDraft(ctx.Request.Context(), id, schoolID, articleType)
	if err != nil {
		return nil, err
	}
	if art.Status == constant.StatusDraft {
		if art.UserID == nil || uint(*art.UserID) != viewerID {
			return nil, ErrArticleNotFoundOrNoPermission
		}
		return art, nil
	}
	if art.PublishStatus == 1 {
		ok, _ := dao.Article().ExistsAndOwnedByWithSchoolAndType(ctx.Request.Context(), id, viewerID, schoolID, articleType)
		if !ok {
			return nil, ErrArticleNotFoundOrNoPermission
		}
	}
	return art, nil
}

// Update 更新，类型不可修改。支持草稿编辑，status 可设为 1 发布
func (s *articleService) Update(ctx *gin.Context, id uint, userID uint, schoolID uint, articleType int, req UpdateArticleReq) error {
	ok, err := dao.Article().ExistsAndOwnedByWithSchoolAndTypeAllowDraft(ctx.Request.Context(), id, userID, schoolID, articleType)
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
	if req.Status != nil && (*req.Status == constant.StatusValid || *req.Status == constant.StatusDraft) {
		updates["status"] = *req.Status
	}
	if req.Images != nil {
		paths := make([]string, len(*req.Images))
		for i, p := range *req.Images {
			paths[i] = oss.PathForStorage(p)
		}
		updates["images"] = pq.StringArray(paths)
		updates["image_count"] = len(paths)
	}
	if len(updates) == 0 {
		return nil
	}
	return dao.Article().UpdateColumns(ctx.Request.Context(), id, updates)
}

// UploadImages 批量上传图片到 OSS，使用雪花 ID 路径，仅返回 URL，不更新文章。草稿也可上传
func (s *articleService) UploadImages(ctx *gin.Context, id uint, userID uint, schoolID uint, articleType int, files []*multipart.FileHeader) ([]string, error) {
	ok, err := dao.Article().ExistsAndOwnedByWithSchoolAndTypeAllowDraft(ctx.Request.Context(), id, userID, schoolID, articleType)
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
	for _, f := range files {
		ext := oss.ExtFromFilename(f.Filename)
		sfID := snowflake.NextID()
		relPath := oss.ArticleImagePathWithSnowflake(id, sfID, ext)
		url, err := oss.Save(f, relPath)
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	return urls, nil
}

// Delete 软删除，类型隔离。草稿也可删除（丢弃）
func (s *articleService) Delete(ctx *gin.Context, id uint, userID uint, schoolID uint, articleType int) error {
	ok, err := dao.Article().ExistsAndOwnedByWithSchoolAndTypeAllowDraft(ctx.Request.Context(), id, userID, schoolID, articleType)
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

// ListDrafts 草稿列表，汇总帖子/提问/回答。type=0 全部 1帖子 2提问 3回答
func (s *articleService) ListDrafts(ctx *gin.Context, userID uint, schoolID uint, articleType int, page, pageSize int) ([]*model.Article, int64, error) {
	if userID == 0 {
		return nil, 0, errors.New("请先登录")
	}
	return dao.Article().ListDrafts(ctx.Request.Context(), userID, schoolID, articleType, page, pageSize)
}

// PublishDraft 草稿发布为正式文章
func (s *articleService) PublishDraft(ctx *gin.Context, id uint, userID uint) error {
	ok, err := dao.Article().PublishDraft(ctx.Request.Context(), id, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrDraftNotFoundOrNoPermission
	}
	return nil
}
