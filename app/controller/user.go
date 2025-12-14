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

type UserController struct {
	userService *service.UserService
}

// 确保 UserController 实现了 UserControllerInterface 接口
var _ UserControllerInterface = (*UserController)(nil)

// NewUserController 创建用户控制器
func NewUserController() *UserController {
	return &UserController{
		userService: service.NewUserService(),
	}
}

// Register 用户注册
func (c *UserController) Register(ctx *gin.Context) {
	var req request.UserRegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	loginResp, err := c.userService.Register(&req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, response.Response{
		Code:    200,
		Message: "注册成功",
		Data:    loginResp,
	})
}

// Login 用户登录
func (c *UserController) Login(ctx *gin.Context) {
	var req request.UserLoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	loginResp, err := c.userService.Login(&req)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, response.Response{
			Code:    401,
			Message: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, response.Response{
		Code:    200,
		Message: "登录成功",
		Data:    loginResp,
	})
}

// GetByID 根据 ID 获取用户
func (c *UserController) GetByID(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "无效的用户ID",
		})
		return
	}

	user, err := c.userService.GetByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, response.Response{
			Code:    404,
			Message: "用户不存在",
		})
		return
	}

	ctx.JSON(http.StatusOK, response.Response{
		Code:    200,
		Message: "获取成功",
		Data:    user,
	})
}

// Update 更新用户信息
func (c *UserController) Update(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "无效的用户ID",
		})
		return
	}

	// 从 JWT 中获取当前用户ID，验证权限
	currentUserID := middleware.GetUserID(ctx)
	if currentUserID != uint(id) {
		ctx.JSON(http.StatusForbidden, response.Response{
			Code:    403,
			Message: "无权限访问此资源",
		})
		return
	}

	var req request.UserUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Code:    400,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	user, err := c.userService.Update(uint(id), &req)
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
		Data:    user,
	})
}

// List 获取用户列表
func (c *UserController) List(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))
	var schoolID *uint
	if sid := ctx.Query("school_id"); sid != "" {
		if id, err := strconv.ParseUint(sid, 10, 32); err == nil {
			uid := uint(id)
			schoolID = &uid
		}
	}

	result, err := c.userService.List(page, pageSize, schoolID)
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

// Info 获取当前登录用户信息
func (c *UserController) Info(ctx *gin.Context) {
	// 从 JWT 中获取当前用户ID
	currentUserID := middleware.GetUserID(ctx)
	if currentUserID == 0 {
		ctx.JSON(http.StatusUnauthorized, response.Response{
			Code:    401,
			Message: "未认证",
		})
		return
	}

	user, err := c.userService.GetCurrentUser(currentUserID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, response.Response{
			Code:    404,
			Message: "用户不存在",
		})
		return
	}

	ctx.JSON(http.StatusOK, response.Response{
		Code:    200,
		Message: "获取成功",
		Data:    user,
	})
}
