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

type LikeController struct {
	likeService *service.LikeService
}

// 确保 LikeController 实现了 LikeControllerInterface 接口
var _ LikeControllerInterface = (*LikeController)(nil)

// NewLikeController 创建点赞控制器
func NewLikeController() *LikeController {
	return &LikeController{
		likeService: service.NewLikeService(),
	}
}

// ToggleLike 切换点赞状态
func (c *LikeController) ToggleLike(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)

	var req request.LikeCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	isLiked, err := c.likeService.ToggleLike(userID, &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	action := "取消点赞"
	if isLiked {
		action = "点赞成功"
	}

	ctx.JSON(http.StatusOK, response.Response{
		Code:    200,
		Message: action,
		Data: gin.H{
			"is_liked": isLiked,
		},
	})
}

// IsLiked 检查是否已点赞
func (c *LikeController) IsLiked(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)

	extTypeStr := ctx.Query("ext_type")
	extIDStr := ctx.Query("ext_id")

	if extTypeStr == "" || extIDStr == "" {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "参数错误: ext_type 和 ext_id 必填",
		})
		return
	}

	extType, err := strconv.Atoi(extTypeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "无效的 ext_type",
		})
		return
	}

	extID, err := strconv.Atoi(extIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "无效的 ext_id",
		})
		return
	}

	isLiked, err := c.likeService.IsLiked(userID, extType, extID)
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
		Data: gin.H{
			"is_liked": isLiked,
		},
	})
}

