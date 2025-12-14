package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/request"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
)

type TagController struct {
	tagService *service.TagService
}

// 确保 TagController 实现了 TagControllerInterface 接口
var _ TagControllerInterface = (*TagController)(nil)

// NewTagController 创建标签控制器
func NewTagController() *TagController {
	return &TagController{
		tagService: service.NewTagService(),
	}
}

// Create 创建标签
func (c *TagController) Create(ctx *gin.Context) {
	var req request.TagCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	tag, err := c.tagService.Create(&req)
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
		Data:    tag,
	})
}

// GetByExt 根据关联对象获取标签列表
func (c *TagController) GetByExt(ctx *gin.Context) {
	extType, err := strconv.Atoi(ctx.Query("ext_type"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "无效的 ext_type",
		})
		return
	}

	extID, err := strconv.Atoi(ctx.Query("ext_id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "无效的 ext_id",
		})
		return
	}

	tags, err := c.tagService.GetByExt(extType, extID)
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
		Data:    tags,
	})
}

