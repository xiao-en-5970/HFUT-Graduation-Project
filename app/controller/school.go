package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/request"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
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
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	school, err := c.schoolService.Create(&req)
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
		Data:    school,
	})
}

// GetByID 根据 ID 获取学校
func (c *SchoolController) GetByID(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "无效的学校ID",
		})
		return
	}

	school, err := c.schoolService.GetByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, response.Response{
			Code:    404,
			Message: "学校不存在",
		})
		return
	}

	ctx.JSON(http.StatusOK, response.Response{
		Code:    200,
		Message: "获取成功",
		Data:    school,
	})
}

// List 获取学校列表
func (c *SchoolController) List(ctx *gin.Context) {
	schools, err := c.schoolService.List()
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
		Data:    schools,
	})
}

