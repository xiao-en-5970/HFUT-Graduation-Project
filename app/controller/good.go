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

type GoodController struct {
	goodService *service.GoodService
}

// 确保 GoodController 实现了 GoodControllerInterface 接口
var _ GoodControllerInterface = (*GoodController)(nil)

// NewGoodController 创建商品控制器
func NewGoodController() *GoodController {
	return &GoodController{
		goodService: service.NewGoodService(),
	}
}

// Create 创建商品
func (c *GoodController) Create(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)

	var req request.GoodCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	good, err := c.goodService.Create(userID, &req)
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
		Data:    good,
	})
}

// GetByID 根据 ID 获取商品
func (c *GoodController) GetByID(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "无效的商品ID",
		})
		return
	}

	good, err := c.goodService.GetByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, response.Response{
			Code:    404,
			Message: "商品不存在",
		})
		return
	}

	ctx.JSON(http.StatusOK, response.Response{
		Code:    200,
		Message: "获取成功",
		Data:    good,
	})
}

// Update 更新商品
func (c *GoodController) Update(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "无效的商品ID",
		})
		return
	}

	userID := middleware.GetUserID(ctx)

	var req request.GoodUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	good, err := c.goodService.Update(userID, uint(id), &req)
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
		Data:    good,
	})
}

// Delete 删除商品
func (c *GoodController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "无效的商品ID",
		})
		return
	}

	userID := middleware.GetUserID(ctx)

	if err := c.goodService.Delete(userID, uint(id)); err != nil {
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

// List 获取商品列表
func (c *GoodController) List(ctx *gin.Context) {
	var req request.GoodListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	result, err := c.goodService.List(&req)
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

