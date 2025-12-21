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

type ArticleController struct {
	articleService *service.ArticleService
}

// 确保 ArticleController 实现了 ArticleControllerInterface 接口
var _ ArticleControllerInterface = (*ArticleController)(nil)

// NewArticleController 创建文章控制器
func NewArticleController() *ArticleController {
	return &ArticleController{
		articleService: service.NewArticleService(),
	}
}

// Create 创建文章
func (c *ArticleController) Create(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)

	var req request.ArticleCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	article, err := c.articleService.Create(userID, &req)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, errcode.ErrArticleCreateFailed, err.Error())
		return
	}

	reply.ReplyOKWithMessageAndData(ctx, "创建成功", article)
}

// GetByID 根据 ID 获取文章
func (c *ArticleController) GetByID(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	article, err := c.articleService.GetByID(uint(id))
	if err != nil {
		reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
		return
	}

	reply.ReplyOKWithData(ctx, article)
}

// Update 更新文章
func (c *ArticleController) Update(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	userID := middleware.GetUserID(ctx)

	var req request.ArticleUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	article, err := c.articleService.Update(userID, uint(id), &req)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, errcode.ErrArticleUpdateFailed, err.Error())
		return
	}

	reply.ReplyOKWithMessageAndData(ctx, "更新成功", article)
}

// Delete 删除文章
func (c *ArticleController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	userID := middleware.GetUserID(ctx)

	if err := c.articleService.Delete(userID, uint(id)); err != nil {
		reply.ReplyErrWithMessage(ctx, errcode.ErrArticleDeleteFailed, err.Error())
		return
	}

	reply.ReplyOKWithMessage(ctx, "删除成功")
}

// List 获取文章列表
func (c *ArticleController) List(ctx *gin.Context) {
	var req request.ArticleListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	result, err := c.articleService.List(&req)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}

	reply.ReplyOKWithData(ctx, result)
}
