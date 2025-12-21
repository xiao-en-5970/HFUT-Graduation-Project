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
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	loginResp, err := c.userService.Register(&req)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, errcode.ErrUserAlreadyExists, err.Error())
		return
	}

	reply.ReplyOKWithMessageAndData(ctx, "注册成功", loginResp)
}

// Login 用户登录
func (c *UserController) Login(ctx *gin.Context) {
	var req request.UserLoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	loginResp, err := c.userService.Login(&req)
	if err != nil {
		reply.ReplyErrWithCodeAndMessage(ctx, 401, errcode.ErrUserPasswordWrong, err.Error())
		return
	}

	reply.ReplyOKWithMessageAndData(ctx, "登录成功", loginResp)
}

// GetByID 根据 ID 获取用户
func (c *UserController) GetByID(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	user, err := c.userService.GetByID(uint(id))
	if err != nil {
		reply.ReplyNotFound(ctx, errcode.ErrUserNotFound)
		return
	}

	reply.ReplyOKWithData(ctx, user)
}

// Update 更新用户信息
func (c *UserController) Update(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	// 从 JWT 中获取当前用户ID，验证权限
	currentUserID := middleware.GetUserID(ctx)
	if currentUserID != uint(id) {
		reply.ReplyForbidden(ctx)
		return
	}

	var req request.UserUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	user, err := c.userService.Update(uint(id), &req)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, errcode.ErrUserNoPermission, err.Error())
		return
	}

	reply.ReplyOKWithMessageAndData(ctx, "更新成功", user)
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
		reply.ReplyInternalError(ctx, err)
		return
	}

	reply.ReplyOKWithData(ctx, result)
}

// Info 获取当前登录用户信息
func (c *UserController) Info(ctx *gin.Context) {
	// 从 JWT 中获取当前用户ID
	currentUserID := middleware.GetUserID(ctx)
	if currentUserID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}

	user, err := c.userService.GetCurrentUser(currentUserID)
	if err != nil {
		reply.ReplyNotFound(ctx, errcode.ErrUserNotFound)
		return
	}

	reply.ReplyOKWithData(ctx, user)
}
