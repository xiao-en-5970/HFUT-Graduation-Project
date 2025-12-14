package service

import (
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/request"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
)

type TagService struct {
	tagDAO *dao.TagDAO
}

// 确保 TagService 实现了 TagServiceInterface 接口
var _ TagServiceInterface = (*TagService)(nil)

// NewTagService 创建标签服务
func NewTagService() *TagService {
	return &TagService{
		tagDAO: dao.NewTagDAO(),
	}
}

// Create 创建标签
func (s *TagService) Create(req *request.TagCreateRequest) (*response.TagResponse, error) {
	tag := &model.Tag{
		Name:    req.Name,
		ExtType: req.ExtType,
		ExtID:   req.ExtID,
		Status:  1,
	}

	if err := s.tagDAO.Create(tag); err != nil {
		return nil, err
	}

	tag, err := s.tagDAO.GetByID(tag.ID)
	if err != nil {
		return nil, err
	}

	return response.ToTagResponse(tag), nil
}

// GetByExt 根据关联对象获取标签列表
func (s *TagService) GetByExt(extType int, extID int) ([]*response.TagResponse, error) {
	tags, err := s.tagDAO.GetByExt(extType, extID)
	if err != nil {
		return nil, err
	}

	var tagResponses []*response.TagResponse
	for _, tag := range tags {
		tagResponses = append(tagResponses, response.ToTagResponse(&tag))
	}

	return tagResponses, nil
}

