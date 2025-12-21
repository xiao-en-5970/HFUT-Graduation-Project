package service

import (
	"errors"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/request"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
)

type OrderService struct {
	orderDAO *dao.OrderDAO
	goodDAO  *dao.GoodDAO
}

// 确保 OrderService 实现了 OrderServiceInterface 接口
var _ OrderServiceInterface = (*OrderService)(nil)

// NewOrderService 创建订单服务
func NewOrderService() *OrderService {
	return &OrderService{
		orderDAO: dao.NewOrderDAO(),
		goodDAO:  dao.NewGoodDAO(),
	}
}

// Create 创建订单
func (s *OrderService) Create(userID uint, req *request.OrderCreateRequest) (*response.OrderResponse, error) {
	// 验证商品是否存在
	good, err := s.goodDAO.GetByID(req.GoodsID)
	if err != nil {
		return nil, errors.New("商品不存在")
	}

	// 检查商品状态
	if good.Status != 1 {
		return nil, errors.New("商品已禁用")
	}
	if good.GoodStatus != 1 {
		return nil, errors.New("商品不在售状态")
	}
	if good.Stock <= 0 {
		return nil, errors.New("商品库存不足")
	}

	// 创建订单
	order := &model.Order{
		UserID:      userID,
		GoodsID:     req.GoodsID,
		OrderStatus: 1, // 默认待支付
		Status:      1, // 正常状态
	}

	if err := s.orderDAO.Create(order); err != nil {
		return nil, err
	}

	// 减少商品库存
	_ = s.goodDAO.DecrementStock(req.GoodsID)

	order, err = s.orderDAO.GetByID(order.ID)
	if err != nil {
		return nil, err
	}

	return response.ToOrderResponse(order), nil
}

// GetByID 根据 ID 获取订单
func (s *OrderService) GetByID(id uint) (*response.OrderResponse, error) {
	order, err := s.orderDAO.GetByID(id)
	if err != nil {
		return nil, err
	}
	return response.ToOrderResponse(order), nil
}

// Update 更新订单
func (s *OrderService) Update(userID, orderID uint, req *request.OrderUpdateRequest) (*response.OrderResponse, error) {
	order, err := s.orderDAO.GetByID(orderID)
	if err != nil {
		return nil, err
	}

	// 检查权限
	if order.UserID != userID {
		return nil, errors.New("无权限修改此订单")
	}

	// 更新字段
	if req.OrderStatus != nil {
		order.OrderStatus = *req.OrderStatus
	}
	if req.Status != nil {
		order.Status = *req.Status
	}

	if err := s.orderDAO.Update(order); err != nil {
		return nil, err
	}

	order, err = s.orderDAO.GetByID(orderID)
	if err != nil {
		return nil, err
	}

	return response.ToOrderResponse(order), nil
}

// Delete 删除订单（软删除）
func (s *OrderService) Delete(userID, orderID uint) error {
	order, err := s.orderDAO.GetByID(orderID)
	if err != nil {
		return err
	}

	// 检查权限
	if order.UserID != userID {
		return errors.New("无权限删除此订单")
	}

	return s.orderDAO.Delete(orderID)
}

// List 获取订单列表
func (s *OrderService) List(req *request.OrderListRequest) (*response.PageResponse, error) {
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	orders, total, err := s.orderDAO.List(req.Page, req.PageSize, req.UserID, req.GoodsID, req.OrderStatus)
	if err != nil {
		return nil, err
	}

	var orderResponses []*response.OrderResponse
	for _, order := range orders {
		orderResponses = append(orderResponses, response.ToOrderResponse(&order))
	}

	return &response.PageResponse{
		List:     orderResponses,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

