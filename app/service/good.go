package service

import (
	"errors"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/request"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
	"github.com/lib/pq"
)

type GoodService struct {
	goodDAO *dao.GoodDAO
}

// 确保 GoodService 实现了 GoodServiceInterface 接口
var _ GoodServiceInterface = (*GoodService)(nil)

// NewGoodService 创建商品服务
func NewGoodService() *GoodService {
	return &GoodService{
		goodDAO: dao.NewGoodDAO(),
	}
}

// Create 创建商品
func (s *GoodService) Create(userID uint, req *request.GoodCreateRequest) (*response.GoodResponse, error) {
	good := &model.Good{
		UserID:     userID,
		Title:      req.Title,
		Content:    req.Content,
		Price:      req.Price,
		Stock:      req.Stock,
		GoodStatus: req.GoodStatus,
		Status:     1,
	}

	if good.GoodStatus == 0 {
		good.GoodStatus = 1 // 默认在售
	}
	if len(req.Images) > 0 {
		good.Images = pq.StringArray(req.Images)
	}

	if err := s.goodDAO.Create(good); err != nil {
		return nil, err
	}

	good, err := s.goodDAO.GetByID(good.ID)
	if err != nil {
		return nil, err
	}

	return response.ToGoodResponse(good), nil
}

// GetByID 根据 ID 获取商品
func (s *GoodService) GetByID(id uint) (*response.GoodResponse, error) {
	good, err := s.goodDAO.GetByID(id)
	if err != nil {
		return nil, err
	}
	return response.ToGoodResponse(good), nil
}

// Update 更新商品
func (s *GoodService) Update(userID, goodID uint, req *request.GoodUpdateRequest) (*response.GoodResponse, error) {
	good, err := s.goodDAO.GetByID(goodID)
	if err != nil {
		return nil, err
	}

	// 检查权限
	if good.UserID != userID {
		return nil, errors.New("无权限修改此商品")
	}

	// 更新字段
	if req.Title != "" {
		good.Title = req.Title
	}
	if req.Content != "" {
		good.Content = req.Content
	}
	if req.Price != nil {
		good.Price = *req.Price
	}
	if req.Stock != nil {
		good.Stock = *req.Stock
	}
	if req.GoodStatus != nil {
		good.GoodStatus = *req.GoodStatus
	}
	if req.Status != nil {
		good.Status = *req.Status
	}
	if len(req.Images) > 0 {
		good.Images = pq.StringArray(req.Images)
	}

	if err := s.goodDAO.Update(good); err != nil {
		return nil, err
	}

	good, err = s.goodDAO.GetByID(goodID)
	if err != nil {
		return nil, err
	}

	return response.ToGoodResponse(good), nil
}

// Delete 删除商品
func (s *GoodService) Delete(userID, goodID uint) error {
	good, err := s.goodDAO.GetByID(goodID)
	if err != nil {
		return err
	}

	// 检查权限
	if good.UserID != userID {
		return errors.New("无权限删除此商品")
	}

	return s.goodDAO.Delete(goodID)
}

// List 获取商品列表
func (s *GoodService) List(req *request.GoodListRequest) (*response.PageResponse, error) {
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	goods, total, err := s.goodDAO.List(req.Page, req.PageSize, req.UserID, req.GoodStatus, req.Status, req.Keyword, req.MinPrice, req.MaxPrice)
	if err != nil {
		return nil, err
	}

	var goodResponses []*response.GoodResponse
	for _, good := range goods {
		goodResponses = append(goodResponses, response.ToGoodResponse(&good))
	}

	return &response.PageResponse{
		List:     goodResponses,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

