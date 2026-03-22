package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service/errno"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/logger"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/schools"
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
		logger.Error(ctx, "用户注册参数错误", zap.Error(err))
		reply.ReplyInvalidParams(ctx, err)
		return
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

// UserSchoolLogin 学校端登录：对接学校 CAS，需验证码的学校传 captcha+captcha_token
// POST /api/v1/user/school-login
// Body: { "school_code": "hfut", "username": "学号", "password": "密码", "captcha": "验证码", "captcha_token": "xxx" }
func UserSchoolLogin(ctx *gin.Context) {
	type Req struct {
		SchoolCode   string `json:"school_code" binding:"required"`
		Username     string `json:"username" binding:"required"`
		Password     string `json:"password" binding:"required"`
		Captcha      string `json:"captcha"`
		CaptchaToken string `json:"captcha_token"`
	}
	var req Req
	if err := ctx.BindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	opts := &schools.LoginOptions{}
	if school, err := dao.School().GetByCode(ctx.Request.Context(), req.SchoolCode); err == nil && school != nil {
		if school.LoginURL != nil {
			opts.LoginURL = *school.LoginURL
		}
		if school.CaptchaURL != nil {
			opts.CaptchaURL = *school.CaptchaURL
		}
	}
	res, err := schools.Login(ctx.Request.Context(), req.SchoolCode, req.Username, req.Password, req.Captcha, req.CaptchaToken, opts)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	if !res.Success {
		reply.ReplyErrWithMessage(ctx, res.Message)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{
		"student_id": res.StudentID,
		"name":       res.Name,
	})
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
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	var req service.UpdateProfileReq
	if err := ctx.BindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if err := service.User().UpdateProfile(ctx, userID, req); err != nil {
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
	var req service.BindSchoolReq
	if err := ctx.BindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	err := service.User().BindSchool(ctx, userID, req)
	if err != nil {
		logger.Error(ctx, "用户绑定学校失败", zap.Error(err))
		msg := err.Error()
		if msg == "" {
			msg = "操作失败，请稍后重试"
		}
		reply.ReplyErrWithMessage(ctx, msg)
		return
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

// UserListPosts 按用户列出帖子 GET /user/:id/posts
func UserListPosts(ctx *gin.Context) {
	UserListArticlesByType(ctx, constant.ArticleTypeNormal)
}

// UserListQuestions 按用户列出提问 GET /user/:id/questions
func UserListQuestions(ctx *gin.Context) {
	UserListArticlesByType(ctx, constant.ArticleTypeQuestion)
}

// UserListAnswers 按用户列出回答 GET /user/:id/answers
func UserListAnswers(ctx *gin.Context) {
	UserListArticlesByType(ctx, constant.ArticleTypeAnswer)
}

// UserListArticlesByType 按用户分页列出指定类型文章，自己看自己含私密，看别人仅公开
func UserListArticlesByType(ctx *gin.Context, articleType int) {
	viewerID := middleware.GetUserID(ctx)
	if viewerID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	idStr := ctx.Param("id")
	targetID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || targetID == 0 {
		reply.ReplyErrWithMessage(ctx, "用户ID无效")
		return
	}
	schoolID := middleware.GetSchoolID(ctx)
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	list, total, err := service.Article().ListByUser(ctx, uint(targetID), viewerID, schoolID, articleType, page, pageSize)
	if err != nil {
		if errors.Is(err, errno.ErrArticleNotFoundOrNoPermission) {
			reply.ReplyNotFound(ctx, errcode.ErrUserNotFound)
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	for _, a := range list {
		a.Images = oss.TransformImageURLs(a.Images)
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": enrichArticlesWithAuthor(ctx, list), "total": total, "page": page, "page_size": pageSize})
}
