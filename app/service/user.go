package service

import (
	"errors"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/request"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/util"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	userDAO   *dao.UserDAO
	schoolDAO *dao.SchoolDAO
}

// 确保 UserService 实现了 UserServiceInterface 接口
var _ UserServiceInterface = (*UserService)(nil)

// NewUserService 创建用户服务
func NewUserService() *UserService {
	return &UserService{
		userDAO:   dao.NewUserDAO(),
		schoolDAO: dao.NewSchoolDAO(),
	}
}

// Register 用户注册
func (s *UserService) Register(req *request.UserRegisterRequest) (*response.LoginResponse, error) {
	// 检查用户名是否已存在
	_, err := s.userDAO.GetByUsername(req.Username)
	if err == nil {
		return nil, errors.New("用户名已存在")
	}
	// 如果错误不是记录不存在，说明是其他错误，应该返回
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 设置默认学校ID为1
	schoolID := uint(1)
	if req.SchoolID != nil {
		schoolID = *req.SchoolID
	}

	// 创建用户
	user := &model.User{
		Username: req.Username,
		Password: string(hashedPassword),
		SchoolID: &schoolID,
		Status:   1,
		Role:     1,
	}

	if err := s.userDAO.Create(user); err != nil {
		return nil, err
	}

	// 增加学校用户数
	_ = s.schoolDAO.IncrementUserCount(schoolID)

	// 获取完整用户信息
	user, err = s.userDAO.GetByID(user.ID)
	if err != nil {
		return nil, err
	}

	// 生成 JWT token
	token, err := util.GenerateToken(user.ID, user.Username)
	if err != nil {
		return nil, errors.New("生成 token 失败: " + err.Error())
	}

	return &response.LoginResponse{
		Token: token,
		User:  *response.ToUserResponse(user),
	}, nil
}

// Login 用户登录
func (s *UserService) Login(req *request.UserLoginRequest) (*response.LoginResponse, error) {
	user, err := s.userDAO.GetByUsername(req.Username)
	if err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	// 验证密码
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	// 检查状态
	if user.Status != 1 {
		return nil, errors.New("账户已被禁用")
	}

	// 生成 JWT token
	token, err := util.GenerateToken(user.ID, user.Username)
	if err != nil {
		return nil, errors.New("生成 token 失败: " + err.Error())
	}

	return &response.LoginResponse{
		Token: token,
		User:  *response.ToUserResponse(user),
	}, nil
}

// GetByID 根据 ID 获取用户
func (s *UserService) GetByID(id uint) (*response.UserResponse, error) {
	user, err := s.userDAO.GetByID(id)
	if err != nil {
		return nil, err
	}
	return response.ToUserResponse(user), nil
}

// Update 更新用户信息
func (s *UserService) Update(id uint, req *request.UserUpdateRequest) (*response.UserResponse, error) {
	user, err := s.userDAO.GetByID(id)
	if err != nil {
		return nil, err
	}

	// 更新字段
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	if req.Background != "" {
		user.Background = req.Background
	}
	if req.BindQQ != "" {
		user.BindQQ = req.BindQQ
	}
	if req.BindWX != "" {
		user.BindWX = req.BindWX
	}
	if req.BindPhone != "" {
		user.BindPhone = req.BindPhone
	}
	if req.SchoolID != nil {
		user.SchoolID = req.SchoolID
	}

	if err := s.userDAO.Update(user); err != nil {
		return nil, err
	}

	user, err = s.userDAO.GetByID(id)
	if err != nil {
		return nil, err
	}

	return response.ToUserResponse(user), nil
}

// List 获取用户列表
func (s *UserService) List(page, pageSize int, schoolID *uint) (*response.PageResponse, error) {
	users, total, err := s.userDAO.List(page, pageSize, schoolID)
	if err != nil {
		return nil, err
	}

	var userResponses []*response.UserResponse
	for _, user := range users {
		userResponses = append(userResponses, response.ToUserResponse(&user))
	}

	return &response.PageResponse{
		List:     userResponses,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

