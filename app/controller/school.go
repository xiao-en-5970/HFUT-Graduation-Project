package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/request"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

type SchoolController struct {
	schoolService *service.SchoolService
}

// 确保 SchoolController 实现了 SchoolControllerInterface 接口
var _ SchoolControllerInterface = (*SchoolController)(nil)

// NewSchoolController 创建学校控制器
func NewSchoolController() *SchoolController {
	return &SchoolController{
		schoolService: service.NewSchoolService(),
	}
}

// Create 创建学校
func (c *SchoolController) Create(ctx *gin.Context) {
	var req request.SchoolCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	school, err := c.schoolService.Create(&req)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, errcode.ErrSchoolCreateFailed, err.Error())
		return
	}

	reply.ReplyOKWithMessageAndData(ctx, "创建成功", school)
}

// GetByID 根据 ID 获取学校
func (c *SchoolController) GetByID(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	school, err := c.schoolService.GetByID(uint(id))
	if err != nil {
		reply.ReplyNotFound(ctx, errcode.ErrSchoolNotFound)
		return
	}

	reply.ReplyOKWithData(ctx, school)
}

// List 获取学校列表
func (c *SchoolController) List(ctx *gin.Context) {
	schools, err := c.schoolService.List()
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}

	reply.ReplyOKWithData(ctx, schools)
}

