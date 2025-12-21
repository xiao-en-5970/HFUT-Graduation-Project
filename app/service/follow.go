package service

import (
	"errors"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/request"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
)

type FollowService struct {
	followDAO *dao.FollowDAO
	userDAO   *dao.UserDAO
}

// 确保 FollowService 实现了 FollowServiceInterface 接口
var _ FollowServiceInterface = (*FollowService)(nil)

// NewFollowService 创建关注服务
func NewFollowService() *FollowService {
	return &FollowService{
		followDAO: dao.NewFollowDAO(),
		userDAO:   dao.NewUserDAO(),
	}
}

// Follow 关注用户
func (s *FollowService) Follow(userID uint, followID uint) (*response.FollowResponse, error) {
	// 不能关注自己
	if userID == followID {
		return nil, errors.New("不能关注自己")
	}

	// 检查被关注用户是否存在
	_, err := s.userDAO.GetByID(followID)
	if err != nil {
		return nil, errors.New("被关注用户不存在")
	}

	// 检查是否已关注
	existing, _ := s.followDAO.GetByUserAndFollow(userID, followID)
	if existing != nil {
		return nil, errors.New("已关注，不能重复关注")
	}

	follow := &model.Follow{
		UserID:   userID,
		FollowID: followID,
	}

	if err := s.followDAO.Create(follow); err != nil {
		return nil, err
	}

	follow, err = s.followDAO.GetByID(follow.ID)
	if err != nil {
		return nil, err
	}

	return response.ToFollowResponse(follow), nil
}

// Unfollow 取消关注
func (s *FollowService) Unfollow(userID uint, followID uint) error {
	// 检查是否已关注
	follow, err := s.followDAO.GetByUserAndFollow(userID, followID)
	if err != nil || follow == nil {
		return errors.New("未关注该用户")
	}

	return s.followDAO.DeleteByUserAndFollow(userID, followID)
}

// GetFollowingList 获取关注列表（我关注的人）
func (s *FollowService) GetFollowingList(req *request.FollowListRequest) (*response.PageResponse, error) {
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	follows, total, err := s.followDAO.ListFollowing(req.Page, req.PageSize, req.UserID)
	if err != nil {
		return nil, err
	}

	var followResponses []*response.FollowResponse
	for _, follow := range follows {
		followResponses = append(followResponses, response.ToFollowResponse(&follow))
	}

	return &response.PageResponse{
		List:     followResponses,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetFollowersList 获取粉丝列表（关注我的人）
func (s *FollowService) GetFollowersList(req *request.FollowListRequest) (*response.PageResponse, error) {
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	follows, total, err := s.followDAO.ListFollowers(req.Page, req.PageSize, req.UserID)
	if err != nil {
		return nil, err
	}

	var followResponses []*response.FollowResponse
	for _, follow := range follows {
		followResponses = append(followResponses, response.ToFollowResponse(&follow))
	}

	return &response.PageResponse{
		List:     followResponses,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetFollowCount 获取关注和粉丝数量
func (s *FollowService) GetFollowCount(userID uint) (*response.FollowCountResponse, error) {
	followingCount, err := s.followDAO.CountFollowing(userID)
	if err != nil {
		return nil, err
	}

	followersCount, err := s.followDAO.CountFollowers(userID)
	if err != nil {
		return nil, err
	}

	return &response.FollowCountResponse{
		FollowingCount: followingCount,
		FollowersCount: followersCount,
	}, nil
}

// IsFollowing 检查是否已关注
func (s *FollowService) IsFollowing(userID uint, followID uint) (bool, error) {
	follow, err := s.followDAO.GetByUserAndFollow(userID, followID)
	if err != nil || follow == nil {
		return false, nil
	}
	return true, nil
}

