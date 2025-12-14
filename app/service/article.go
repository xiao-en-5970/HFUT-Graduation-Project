package service

import (
	"errors"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/request"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
)

type ArticleService struct {
	articleDAO *dao.ArticleDAO
}

// 确保 ArticleService 实现了 ArticleServiceInterface 接口
var _ ArticleServiceInterface = (*ArticleService)(nil)

// NewArticleService 创建文章服务
func NewArticleService() *ArticleService {
	return &ArticleService{
		articleDAO: dao.NewArticleDAO(),
	}
}

// Create 创建文章
func (s *ArticleService) Create(userID uint, req *request.ArticleCreateRequest) (*response.ArticleResponse, error) {
	article := &model.Article{
		UserID:        userID,
		Title:         req.Title,
		Content:       req.Content,
		PublishStatus: req.PublishStatus,
		Type:          req.Type,
		Status:        1,
		ViewCount:     0,
		LikeCount:     0,
		CollectCount:  0,
	}

	if article.PublishStatus == 0 {
		article.PublishStatus = 1 // 默认私密
	}
	if article.Type == 0 {
		article.Type = 1 // 默认普通文章
	}

	if err := s.articleDAO.Create(article); err != nil {
		return nil, err
	}

	article, err := s.articleDAO.GetByID(article.ID)
	if err != nil {
		return nil, err
	}

	return response.ToArticleResponse(article), nil
}

// GetByID 根据 ID 获取文章
func (s *ArticleService) GetByID(id uint) (*response.ArticleResponse, error) {
	article, err := s.articleDAO.GetByID(id)
	if err != nil {
		return nil, err
	}

	// 增加浏览次数
	_ = s.articleDAO.IncrementViewCount(id)

	// 重新获取以获取更新后的数据
	updatedArticle, err := s.articleDAO.GetByID(id)
	if err == nil && updatedArticle != nil {
		article = updatedArticle
	}

	return response.ToArticleResponse(article), nil
}

// Update 更新文章
func (s *ArticleService) Update(userID, articleID uint, req *request.ArticleUpdateRequest) (*response.ArticleResponse, error) {
	article, err := s.articleDAO.GetByID(articleID)
	if err != nil {
		return nil, err
	}

	// 检查权限
	if article.UserID != userID {
		return nil, errors.New("无权限修改此文章")
	}

	// 更新字段
	if req.Title != "" {
		article.Title = req.Title
	}
	if req.Content != "" {
		article.Content = req.Content
	}
	if req.PublishStatus != nil {
		article.PublishStatus = *req.PublishStatus
	}
	if req.Status != nil {
		article.Status = *req.Status
	}

	if err := s.articleDAO.Update(article); err != nil {
		return nil, err
	}

	article, err = s.articleDAO.GetByID(articleID)
	if err != nil {
		return nil, err
	}

	return response.ToArticleResponse(article), nil
}

// Delete 删除文章
func (s *ArticleService) Delete(userID, articleID uint) error {
	article, err := s.articleDAO.GetByID(articleID)
	if err != nil {
		return err
	}

	// 检查权限
	if article.UserID != userID {
		return errors.New("无权限删除此文章")
	}

	return s.articleDAO.Delete(articleID)
}

// List 获取文章列表
func (s *ArticleService) List(req *request.ArticleListRequest) (*response.PageResponse, error) {
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	articles, total, err := s.articleDAO.List(req.Page, req.PageSize, req.UserID, req.Type, req.Status, req.Keyword)
	if err != nil {
		return nil, err
	}

	var articleResponses []*response.ArticleResponse
	for _, article := range articles {
		articleResponses = append(articleResponses, response.ToArticleResponse(&article))
	}

	return &response.PageResponse{
		List:     articleResponses,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

