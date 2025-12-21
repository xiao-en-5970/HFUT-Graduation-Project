package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/request"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
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
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	tag, err := c.tagService.Create(&req)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, errcode.ErrTagCreateFailed, err.Error())
		return
	}

	reply.ReplyOKWithMessageAndData(ctx, "创建成功", tag)
}

// GetByExt 根据关联对象获取标签列表
func (c *TagController) GetByExt(ctx *gin.Context) {
	extType, err := strconv.Atoi(ctx.Query("ext_type"))
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	extID, err := strconv.Atoi(ctx.Query("ext_id"))
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	tags, err := c.tagService.GetByExt(extType, extID)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}

	reply.ReplyOKWithData(ctx, tags)
}

