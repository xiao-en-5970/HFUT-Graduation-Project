package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// AdminLogin 管理员登录：账号密码，仅 role>=2 可登录
func AdminLogin(ctx *gin.Context) {
	var body struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	token, err := service.User().AdminLogin(ctx, body.Username, body.Password)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOKWithMessageAndData(ctx, "登录成功", gin.H{"token": token})
}

// AdminUserList 管理员：用户列表
func AdminUserList(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	statusFilter, _ := strconv.Atoi(ctx.DefaultQuery("status", "0"))
	list, total, err := dao.User().List(ctx.Request.Context(), page, pageSize, int16(statusFilter))
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	// 不返回密码
	for _, u := range list {
		if u != nil {
			u.Password = ""
		}
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": list, "total": total, "page": page, "page_size": pageSize})
}

// AdminUserDisable 管理员：禁用用户
func AdminUserDisable(ctx *gin.Context) {
	id, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	if err := dao.User().UpdateStatus(ctx.Request.Context(), id, constant.StatusInvalid); err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// AdminUserRestore 管理员：恢复用户
func AdminUserRestore(ctx *gin.Context) {
	id, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	if err := dao.User().UpdateStatus(ctx.Request.Context(), id, constant.StatusValid); err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// AdminUserUpdateStatus 管理员：修改用户状态（启用/禁用）
func AdminUserUpdateStatus(ctx *gin.Context) {
	id, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	var body struct {
		Status int16 `json:"status" binding:"required"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if body.Status != constant.StatusValid && body.Status != constant.StatusInvalid {
		reply.ReplyErrWithMessage(ctx, "status 无效，1正常 2禁用")
		return
	}
	if err := dao.User().UpdateStatus(ctx.Request.Context(), id, body.Status); err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// AdminUserUpdateRole 管理员：修改用户角色
func AdminUserUpdateRole(ctx *gin.Context) {
	id, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	var body struct {
		Role int16 `json:"role" binding:"required"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if body.Role < constant.RoleUser || body.Role > constant.RoleAnonymous {
		reply.ReplyErrWithMessage(ctx, "角色值无效，1普通 2管理员 3超级管理员 4匿名")
		return
	}
	if err := dao.User().UpdateRole(ctx.Request.Context(), id, body.Role); err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// AdminPostDisable 管理员：禁用帖子
func AdminPostDisable(ctx *gin.Context) {
	adminArticleDisable(ctx, constant.ArticleTypeNormal)
}

// AdminPostRestore 管理员：恢复帖子
func AdminPostRestore(ctx *gin.Context) {
	adminArticleRestore(ctx, constant.ArticleTypeNormal)
}

// AdminQuestionDisable 管理员：禁用提问
func AdminQuestionDisable(ctx *gin.Context) {
	adminArticleDisable(ctx, constant.ArticleTypeQuestion)
}

// AdminQuestionRestore 管理员：恢复提问
func AdminQuestionRestore(ctx *gin.Context) {
	adminArticleRestore(ctx, constant.ArticleTypeQuestion)
}

// AdminAnswerDisable 管理员：禁用回答
func AdminAnswerDisable(ctx *gin.Context) {
	adminArticleDisable(ctx, constant.ArticleTypeAnswer)
}

// AdminAnswerRestore 管理员：恢复回答
func AdminAnswerRestore(ctx *gin.Context) {
	adminArticleRestore(ctx, constant.ArticleTypeAnswer)
}

func adminArticleDisable(ctx *gin.Context, articleType int) {
	id, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	_, err := dao.Article().GetByIDIncludeDeleted(ctx.Request.Context(), id)
	if err != nil {
		reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
		return
	}
	if err := dao.Article().SoftDelete(ctx.Request.Context(), id); err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

func adminArticleRestore(ctx *gin.Context, articleType int) {
	id, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	if err := dao.Article().Restore(ctx.Request.Context(), id); err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// AdminArticleList 管理员：文章列表（帖子/提问/回答），include_invalid=1 可含已删除
func AdminArticleList(ctx *gin.Context, articleType int) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	schoolID, _ := strconv.ParseUint(ctx.DefaultQuery("school_id", "0"), 10, 32)
	includeInvalid := ctx.Query("include_invalid") == "1"
	list, total, err := dao.Article().ListAdmin(ctx.Request.Context(), uint(schoolID), articleType, includeInvalid, page, pageSize)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": list, "total": total, "page": page, "page_size": pageSize})
}

// AdminSchoolList 管理员：学校列表
func AdminSchoolList(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	includeInvalid := ctx.Query("include_invalid") == "1"
	list, total, err := dao.School().List(ctx.Request.Context(), page, pageSize, includeInvalid)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": list, "total": total, "page": page, "page_size": pageSize})
}

// AdminSchoolCreate 管理员：新增学校
func AdminSchoolCreate(ctx *gin.Context) {
	var body struct {
		Name     string `json:"name"`
		LoginURL string `json:"login_url"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	school := &model.School{
		Name:     strPtr(body.Name),
		LoginURL: strPtr(body.LoginURL),
		Status:   constant.StatusValid,
	}
	id, err := dao.School().Create(ctx.Request.Context(), school)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"id": id})
}

// AdminSchoolDisable 管理员：下架学校
func AdminSchoolDisable(ctx *gin.Context) {
	id, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	if err := dao.School().UpdateStatus(ctx.Request.Context(), id, constant.StatusInvalid); err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// AdminSchoolRestore 管理员：恢复学校
func AdminSchoolRestore(ctx *gin.Context) {
	id, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	if err := dao.School().UpdateStatus(ctx.Request.Context(), id, constant.StatusValid); err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

func parseID(ctx *gin.Context, param string) (uint, bool) {
	s := ctx.Param(param)
	id, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, "ID 格式无效")
		return 0, false
	}
	return uint(id), true
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
