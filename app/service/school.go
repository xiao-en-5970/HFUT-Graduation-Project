package service

import (
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/request"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
)

type SchoolService struct {
	schoolDAO *dao.SchoolDAO
}

// 确保 SchoolService 实现了 SchoolServiceInterface 接口
var _ SchoolServiceInterface = (*SchoolService)(nil)

// NewSchoolService 创建学校服务
func NewSchoolService() *SchoolService {
	return &SchoolService{
		schoolDAO: dao.NewSchoolDAO(),
	}
}

// Create 创建学校
func (s *SchoolService) Create(req *request.SchoolCreateRequest) (*response.SchoolResponse, error) {
	school := &model.School{
		Name:     req.Name,
		LoginURL: req.LoginURL,
		UserCount: 0,
	}

	if err := s.schoolDAO.Create(school); err != nil {
		return nil, err
	}

	school, err := s.schoolDAO.GetByID(school.ID)
	if err != nil {
		return nil, err
	}

	return response.ToSchoolResponse(school), nil
}

// GetByID 根据 ID 获取学校
func (s *SchoolService) GetByID(id uint) (*response.SchoolResponse, error) {
	school, err := s.schoolDAO.GetByID(id)
	if err != nil {
		return nil, err
	}
	return response.ToSchoolResponse(school), nil
}

// List 获取学校列表
func (s *SchoolService) List() ([]*response.SchoolResponse, error) {
	schools, err := s.schoolDAO.List()
	if err != nil {
		return nil, err
	}

	var schoolResponses []*response.SchoolResponse
	for _, school := range schools {
		schoolResponses = append(schoolResponses, response.ToSchoolResponse(&school))
	}

	return schoolResponses, nil
}

