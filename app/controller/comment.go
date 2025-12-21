package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/request"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

type CommentController struct {
	commentService *service.CommentService
}

// 确保 CommentController 实现了 CommentControllerInterface 接口
var _ CommentControllerInterface = (*CommentController)(nil)

// NewCommentController 创建评论控制器
func NewCommentController() *CommentController {
	return &CommentController{
		commentService: service.NewCommentService(),
	}
}

// Create 创建评论
func (c *CommentController) Create(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)

	var req request.CommentCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	comment, err := c.commentService.Create(userID, &req)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, errcode.ErrCommentCreateFailed, err.Error())
		return
	}

	reply.ReplyOKWithMessageAndData(ctx, "评论成功", comment)
}

// List 获取评论列表
func (c *CommentController) List(ctx *gin.Context) {
	var req request.CommentListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	result, err := c.commentService.List(&req)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}

	reply.ReplyOKWithData(ctx, result)
}

// Delete 删除评论
func (c *CommentController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	userID := middleware.GetUserID(ctx)

	if err := c.commentService.Delete(userID, uint(id)); err != nil {
		reply.ReplyErrWithMessage(ctx, errcode.ErrCommentNoPermission, err.Error())
		return
	}

	reply.ReplyOKWithMessage(ctx, "删除成功")
}
