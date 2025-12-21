package service

import (
	"errors"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/request"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
)

type CollectService struct {
	collectDAO *dao.CollectDAO
	articleDAO *dao.ArticleDAO
	goodDAO    *dao.GoodDAO
}

// 确保 CollectService 实现了 CollectServiceInterface 接口
var _ CollectServiceInterface = (*CollectService)(nil)

// NewCollectService 创建收藏服务
func NewCollectService() *CollectService {
	return &CollectService{
		collectDAO: dao.NewCollectDAO(),
		articleDAO: dao.NewArticleDAO(),
		goodDAO:    dao.NewGoodDAO(),
	}
}

// Create 创建收藏
func (s *CollectService) Create(userID uint, req *request.CollectCreateRequest) (*response.CollectResponse, error) {
	// 检查是否已收藏
	existing, _ := s.collectDAO.GetByUserAndExt(userID, req.ExtType, req.ExtID)
	if existing != nil {
		return nil, errors.New("已收藏，不能重复收藏")
	}

	// 验证关联对象是否存在
	if err := s.validateExt(req.ExtType, req.ExtID); err != nil {
		return nil, err
	}

	collect := &model.Collect{
		UserID:  userID,
		ExtType: req.ExtType,
		ExtID:   req.ExtID,
	}

	if err := s.collectDAO.Create(collect); err != nil {
		return nil, err
	}

	// 更新关联对象的收藏数
	s.incrementCollectCount(req.ExtType, req.ExtID)

	collect, err := s.collectDAO.GetByID(collect.ID)
	if err != nil {
		return nil, err
	}

	return response.ToCollectResponse(collect), nil
}

// GetByID 根据 ID 获取收藏
func (s *CollectService) GetByID(id uint) (*response.CollectResponse, error) {
	collect, err := s.collectDAO.GetByID(id)
	if err != nil {
		return nil, err
	}
	return response.ToCollectResponse(collect), nil
}

// Delete 删除收藏
func (s *CollectService) Delete(userID, collectID uint) error {
	collect, err := s.collectDAO.GetByID(collectID)
	if err != nil {
		return err
	}

	// 检查权限
	if collect.UserID != userID {
		return errors.New("无权限删除此收藏")
	}

	// 更新关联对象的收藏数
	s.decrementCollectCount(collect.ExtType, collect.ExtID)

	return s.collectDAO.Delete(collectID)
}

// DeleteByExt 根据关联对象删除收藏
func (s *CollectService) DeleteByExt(userID uint, req *request.CollectDeleteRequest) error {
	// 检查是否已收藏
	collect, err := s.collectDAO.GetByUserAndExt(userID, req.ExtType, req.ExtID)
	if err != nil || collect == nil {
		return errors.New("未收藏")
	}

	// 更新关联对象的收藏数
	s.decrementCollectCount(req.ExtType, req.ExtID)

	return s.collectDAO.DeleteByUserAndExt(userID, req.ExtType, req.ExtID)
}

// List 获取收藏列表
func (s *CollectService) List(req *request.CollectListRequest) (*response.PageResponse, error) {
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	collects, total, err := s.collectDAO.List(req.Page, req.PageSize, req.UserID, req.ExtType, req.ExtID)
	if err != nil {
		return nil, err
	}

	var collectResponses []*response.CollectResponse
	for _, collect := range collects {
		collectResponses = append(collectResponses, response.ToCollectResponse(&collect))
	}

	return &response.PageResponse{
		List:     collectResponses,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// IsCollected 检查是否已收藏
func (s *CollectService) IsCollected(userID uint, extType int, extID int) (bool, error) {
	collect, err := s.collectDAO.GetByUserAndExt(userID, extType, extID)
	if err != nil || collect == nil {
		return false, nil
	}
	return true, nil
}

// validateExt 验证关联对象是否存在
func (s *CollectService) validateExt(extType int, extID int) error {
	switch extType {
	case 1: // articles
		_, err := s.articleDAO.GetByID(uint(extID))
		if err != nil {
			return errors.New("文章不存在")
		}
	case 2: // goods
		_, err := s.goodDAO.GetByID(uint(extID))
		if err != nil {
			return errors.New("商品不存在")
		}
	default:
		return errors.New("无效的关联类型")
	}
	return nil
}

// incrementCollectCount 增加关联对象的收藏数
func (s *CollectService) incrementCollectCount(extType int, extID int) {
	switch extType {
	case 1: // articles
		_ = s.articleDAO.IncrementCollectCount(uint(extID))
	case 2: // goods
		// 商品表可能没有收藏数字段，这里先不处理
	}
}

// decrementCollectCount 减少关联对象的收藏数
func (s *CollectService) decrementCollectCount(extType int, extID int) {
	switch extType {
	case 1: // articles
		_ = s.articleDAO.DecrementCollectCount(uint(extID))
	case 2: // goods
		// 商品表可能没有收藏数字段，这里先不处理
	}
}

