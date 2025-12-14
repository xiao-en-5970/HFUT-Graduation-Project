package service

import (
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/request"
	"github.com/lib/pq"
)

type LikeService struct {
	likeDAO    *dao.LikeDAO
	articleDAO *dao.ArticleDAO
	commentDAO *dao.CommentDAO
}

// 确保 LikeService 实现了 LikeServiceInterface 接口
var _ LikeServiceInterface = (*LikeService)(nil)

// NewLikeService 创建点赞服务
func NewLikeService() *LikeService {
	return &LikeService{
		likeDAO:    dao.NewLikeDAO(),
		articleDAO: dao.NewArticleDAO(),
		commentDAO: dao.NewCommentDAO(),
	}
}

// ToggleLike 切换点赞状态
func (s *LikeService) ToggleLike(userID uint, req *request.LikeCreateRequest) (bool, error) {
	// 检查是否已点赞
	like, err := s.likeDAO.GetByUserAndExt(userID, req.ExtType, req.ExtID)
	if err == nil && like != nil && like.Status == 1 {
		// 取消点赞
		if err := s.likeDAO.Delete(userID, req.ExtType, req.ExtID); err != nil {
			return false, err
		}
		// 更新关联对象的点赞数
		s.decrementLikeCount(req.ExtType, req.ExtID)
		return false, nil
	}

	// 创建点赞
	newLike := &model.Like{
		UserID:  userID,
		ExtType: req.ExtType,
		ExtID:   req.ExtID,
		Status:  1,
	}
	if len(req.Images) > 0 {
		newLike.Images = pq.StringArray(req.Images)
	}

	if err := s.likeDAO.Create(newLike); err != nil {
		return false, err
	}

	// 更新关联对象的点赞数
	s.incrementLikeCount(req.ExtType, req.ExtID)
	return true, nil
}

// incrementLikeCount 增加关联对象的点赞数
func (s *LikeService) incrementLikeCount(extType int, extID int) {
	switch extType {
	case 1: // articles
		_ = s.articleDAO.IncrementLikeCount(uint(extID))
	case 2: // comments
		_ = s.commentDAO.IncrementLikeCount(uint(extID))
	}
}

// decrementLikeCount 减少关联对象的点赞数
func (s *LikeService) decrementLikeCount(extType int, extID int) {
	switch extType {
	case 1: // articles
		_ = s.articleDAO.DecrementLikeCount(uint(extID))
	case 2: // comments
		_ = s.commentDAO.DecrementLikeCount(uint(extID))
	}
}

// IsLiked 检查是否已点赞
func (s *LikeService) IsLiked(userID uint, extType int, extID int) (bool, error) {
	like, err := s.likeDAO.GetByUserAndExt(userID, extType, extID)
	if err != nil || like == nil {
		return false, nil
	}
	return like.Status == 1, nil
}

