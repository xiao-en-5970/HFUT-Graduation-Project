package service

import (
	"errors"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/request"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
	"github.com/lib/pq"
)

type CommentService struct {
	commentDAO *dao.CommentDAO
}

// 确保 CommentService 实现了 CommentServiceInterface 接口
var _ CommentServiceInterface = (*CommentService)(nil)

// NewCommentService 创建评论服务
func NewCommentService() *CommentService {
	return &CommentService{
		commentDAO: dao.NewCommentDAO(),
	}
}

// Create 创建评论
func (s *CommentService) Create(userID uint, req *request.CommentCreateRequest) (*response.CommentResponse, error) {
	commentType := 1 // 顶层评论
	if req.ParentID != nil {
		commentType = 2 // 回复评论
	}

	comment := &model.Comment{
		UserID:  userID,
		ExtType: req.ExtType,
		ExtID:   req.ExtID,
		Type:    commentType,
		Content: req.Content,
		Status:  1,
		LikeCount: 0,
	}

	if req.ParentID != nil {
		comment.ParentID = req.ParentID
	}
	if req.ReplyID != nil {
		comment.ReplyID = req.ReplyID
	}
	if len(req.Images) > 0 {
		comment.Images = pq.StringArray(req.Images)
	}

	if err := s.commentDAO.Create(comment); err != nil {
		return nil, err
	}

	comment, err := s.commentDAO.GetByID(comment.ID)
	if err != nil {
		return nil, err
	}

	return response.ToCommentResponse(comment), nil
}

// List 获取评论列表
func (s *CommentService) List(req *request.CommentListRequest) (*response.PageResponse, error) {
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	comments, total, err := s.commentDAO.List(req.Page, req.PageSize, req.ExtType, req.ExtID)
	if err != nil {
		return nil, err
	}

	var commentResponses []response.CommentResponse
	for _, comment := range comments {
		commentResp := response.ToCommentResponse(&comment)
		// 获取回复列表
		replies, _ := s.commentDAO.GetReplies(comment.ID)
		var replyResponses []response.CommentResponse
		for _, reply := range replies {
			replyResponses = append(replyResponses, *response.ToCommentResponse(&reply))
		}
		commentResp.Replies = replyResponses
		commentResponses = append(commentResponses, *commentResp)
	}

	return &response.PageResponse{
		List:     commentResponses,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// Delete 删除评论
func (s *CommentService) Delete(userID, commentID uint) error {
	comment, err := s.commentDAO.GetByID(commentID)
	if err != nil {
		return err
	}

	// 检查权限
	if comment.UserID != userID {
		return errors.New("无权限删除此评论")
	}

	return s.commentDAO.Delete(commentID)
}

