package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/request"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
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
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	article, err := c.articleService.Create(userID, &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, response.Response{
		Code:    200,
		Message: "创建成功",
		Data:    article,
	})
}

// GetByID 根据 ID 获取文章
func (c *ArticleController) GetByID(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "无效的文章ID",
		})
		return
	}

	article, err := c.articleService.GetByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, response.Response{
			Code:    404,
			Message: "文章不存在",
		})
		return
	}

	ctx.JSON(http.StatusOK, response.Response{
		Code:    200,
		Message: "获取成功",
		Data:    article,
	})
}

// Update 更新文章
func (c *ArticleController) Update(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "无效的文章ID",
		})
		return
	}

	userID := middleware.GetUserID(ctx)

	var req request.ArticleUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	article, err := c.articleService.Update(userID, uint(id), &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, response.Response{
		Code:    200,
		Message: "更新成功",
		Data:    article,
	})
}

// Delete 删除文章
func (c *ArticleController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "无效的文章ID",
		})
		return
	}

	userID := middleware.GetUserID(ctx)

	if err := c.articleService.Delete(userID, uint(id)); err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, response.Response{
		Code:    200,
		Message: "删除成功",
	})
}

// List 获取文章列表
func (c *ArticleController) List(ctx *gin.Context) {
	var req request.ArticleListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	result, err := c.articleService.List(&req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, response.Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, response.Response{
		Code:    200,
		Message: "获取成功",
		Data:    result,
	})
}

