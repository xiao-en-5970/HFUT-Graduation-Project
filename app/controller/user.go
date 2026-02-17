package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/logger"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
	"go.uber.org/zap"
)

func UserRegister(ctx *gin.Context) {
	type Register struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		RePassword string `json:"re_password"`
	}
	var user Register
	if err := ctx.BindJSON(&user); err != nil {
		logger.Error(ctx, "用户注册失败", zap.Error(err))
	}
	if user.RePassword != user.Password {
		logger.Errorf(ctx, "两次密码不一致！")
	}
	service.User().Register(ctx, user.Username, user.Password)
	reply.ReplyOK(ctx)
}

// Login 用户登录
func UserLogin(ctx *gin.Context) {
	type Login struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var user Login
	if err := ctx.BindJSON(&user); err != nil {
		logger.Error(ctx, "用户登录失败", zap.Error(err))
	}
	token, err := service.User().Login(ctx, user.Username, user.Password)
	if err != nil {
		logger.Error(ctx, "用户登录失败", zap.Error(err))
	}
	reply.ReplyOKWithMessageAndData(ctx, "登录成功", token)
}

func UserInfo(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	info, err := service.User().Info(ctx, userID)
	if err != nil {
		logger.Error(ctx, "用户信息获取失败", zap.Error(err))
	}
	reply.ReplyOKWithData(ctx, info)
}

func UserUpdate(ctx *gin.Context) {
	user := &model.User{}
	err := ctx.BindJSON(user)
	if err != nil {
		logger.Error(ctx, "用户信息更新失败", zap.Error(err))
	}
	service.User().Update(ctx, user)
}

func UserBindSchool(ctx *gin.Context) {
	schoolId := uint(0)
	ctx.BindJSON(&schoolId)
	err := service.User().BindSchool(ctx, schoolId)
	if err != nil {
		logger.Error(ctx, "用户绑定学校失败", zap.Error(err))
	}
	reply.ReplyOK(ctx)
}

func UserLogout(ctx *gin.Context) {
	return
}
