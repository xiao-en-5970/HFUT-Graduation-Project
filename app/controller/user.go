package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/logger"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
	"go.uber.org/zap"
	"gorm.io/gorm"
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
		reply.ReplyErrWithMessage(ctx, "两次密码不一致！")
		return
	}
	userId, err := service.User().Register(ctx, user.Username, user.Password)
	if err != nil {
		reply.ReplyErr(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"user_id": userId})
}

// Login 用户登录
func UserLogin(ctx *gin.Context) {
	type Login struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var user Login
	if err := ctx.BindJSON(&user); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	token, err := service.User().Login(ctx, user.Username, user.Password)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithMessageAndData(ctx, "登录成功", token)
	return
}

func UserInfo(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyErrWithMessage(ctx, "用户不存在")
		return
	}
	info, err := service.User().Info(ctx, userID)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, info)
}

// UserProfile 获取任意非删用户的公开身份信息（需 JWT）
func UserProfile(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		reply.ReplyErrWithMessage(ctx, "用户ID无效")
		return
	}
	profile, err := service.User().GetProfile(ctx, uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			reply.ReplyNotFound(ctx, errcode.ErrUserNotFound)
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, profile)
}

func UserUpdate(ctx *gin.Context) {
	user := &model.User{}
	err := ctx.BindJSON(user)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	err = service.User().Update(ctx, user)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

func UserBindSchool(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	schoolId := uint(0)
	err := ctx.BindJSON(&schoolId)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	err = service.User().BindSchool(ctx, userID, schoolId)
	if err != nil {
		logger.Error(ctx, "用户绑定学校失败", zap.Error(err))
		reply.ReplyInternalError(ctx, err)
	}
	reply.ReplyOK(ctx)
}

func UserLogout(ctx *gin.Context) {
	return
}

func UserUploadAvatar(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	file, err := ctx.FormFile("file")
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	url, err := service.User().UploadAvatar(ctx, userID, file)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"url": url})
}

func UserUploadBackground(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	file, err := ctx.FormFile("file")
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	url, err := service.User().UploadBackground(ctx, userID, file)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"url": url})
}
