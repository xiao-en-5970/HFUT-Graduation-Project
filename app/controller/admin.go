package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
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

// AdminUserCreate 管理员：创建用户
func AdminUserCreate(ctx *gin.Context) {
	var body struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		SchoolID uint   `json:"school_id"`
		Role     int16  `json:"role"`
		Status   int16  `json:"status"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	_, err := dao.User().GetByUsername(ctx.Request.Context(), body.Username)
	if err == nil {
		reply.ReplyErrWithMessage(ctx, "用户名已存在")
		return
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		reply.ReplyInternalError(ctx, err)
		return
	}
	if body.Role == 0 {
		body.Role = constant.RoleUser
	}
	if body.Status == 0 {
		body.Status = constant.StatusValid
	}
	if body.Role < constant.RoleUser || body.Role > constant.RoleAnonymous {
		reply.ReplyErrWithMessage(ctx, "角色值无效，1普通 2管理员 3超管 4匿名")
		return
	}
	if body.Status != constant.StatusValid && body.Status != constant.StatusInvalid {
		reply.ReplyErrWithMessage(ctx, "状态值无效，1正常 2禁用")
		return
	}
	if body.SchoolID > 0 {
		_, err := dao.School().GetByID(ctx.Request.Context(), body.SchoolID)
		if err != nil {
			reply.ReplyErrWithMessage(ctx, "学校不存在")
			return
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	user := &model.User{
		Username: body.Username,
		Password: string(hash),
		SchoolID: body.SchoolID,
		Role:     body.Role,
		Status:   body.Status,
	}
	id, err := dao.User().Create(ctx.Request.Context(), user)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"id": id})
}

// AdminUserUpdate 管理员：修改用户基本信息（不含 role/status，另有专用接口）
func AdminUserUpdate(ctx *gin.Context) {
	id, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	if _, err := dao.User().GetByID(ctx.Request.Context(), id); err != nil {
		reply.ReplyErrWithMessage(ctx, "用户不存在")
		return
	}
	var body struct {
		SchoolID   *uint   `json:"school_id"`
		Avatar     *string `json:"avatar"`     // 头像 URL，OSS 上传后传入
		Background *string `json:"background"` // 背景图 URL
	}
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	updates := make(map[string]interface{})
	if body.SchoolID != nil {
		if *body.SchoolID > 0 {
			_, err := dao.School().GetByID(ctx.Request.Context(), *body.SchoolID)
			if err != nil {
				reply.ReplyErrWithMessage(ctx, "学校不存在")
				return
			}
			updates["school_id"] = *body.SchoolID
		} else {
			updates["school_id"] = nil
		}
	}
	if body.Avatar != nil {
		updates["avatar"] = oss.PathForStorage(*body.Avatar)
	}
	if body.Background != nil {
		updates["background"] = oss.PathForStorage(*body.Background)
	}
	if len(updates) > 0 {
		if err := dao.User().UpdateColumns(ctx.Request.Context(), id, updates); err != nil {
			reply.ReplyInternalError(ctx, err)
			return
		}
	}
	reply.ReplyOK(ctx)
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
	// 不返回密码，将头像/背景转为完整 URL
	for _, u := range list {
		if u != nil {
			u.Password = ""
			u.Avatar = oss.ToFullURL(u.Avatar)
			u.Background = oss.ToFullURL(u.Background)
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

// AdminArticleCreate 管理员：创建草稿（status=3），仅存元信息，返回 id 供后续编辑和图片上传
func AdminArticleCreate(ctx *gin.Context, articleType int) {
	var body struct {
		Title         string `json:"title"`
		Content       string `json:"content"`
		UserID        *uint  `json:"user_id"`
		SchoolID      *uint  `json:"school_id"`
		PublishStatus int16  `json:"publish_status"`
		ParentID      *uint  `json:"parent_id"` // 回答必传
	}
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	var parent *model.Article
	if articleType == constant.ArticleTypeAnswer {
		if body.ParentID == nil || *body.ParentID == 0 {
			reply.ReplyErrWithMessage(ctx, "回答必须指定 parent_id（提问ID）")
			return
		}
		var err error
		parent, err = dao.Article().GetByIDIncludeDeleted(ctx.Request.Context(), *body.ParentID)
		if err != nil || parent == nil || parent.Type != constant.ArticleTypeQuestion {
			reply.ReplyErrWithMessage(ctx, "父提问不存在")
			return
		}
	}
	var uid, sid, pid *int
	if body.UserID != nil && *body.UserID > 0 {
		_, err := dao.User().GetByID(ctx.Request.Context(), *body.UserID)
		if err != nil {
			reply.ReplyErrWithMessage(ctx, "用户不存在")
			return
		}
		u := int(*body.UserID)
		uid = &u
	}
	if body.SchoolID != nil {
		if *body.SchoolID > 0 {
			_, err := dao.School().GetByID(ctx.Request.Context(), *body.SchoolID)
			if err != nil {
				reply.ReplyErrWithMessage(ctx, "学校不存在")
				return
			}
			s := int(*body.SchoolID)
			sid = &s
		} else {
			// *body.SchoolID == 0 表示全站公开
			zero := 0
			sid = &zero
		}
	}
	if articleType == constant.ArticleTypeAnswer && body.ParentID != nil {
		p := int(*body.ParentID)
		pid = &p
		if sid == nil && parent != nil {
			if parent.SchoolID != nil {
				sid = parent.SchoolID
			} else {
				zero := 0
				sid = &zero
			}
		}
	}
	pubStatus := body.PublishStatus
	if pubStatus == 0 {
		pubStatus = 2
	}
	a := &model.Article{
		UserID:        uid,
		SchoolID:      sid,
		ParentID:      pid,
		Title:         body.Title,
		Content:       body.Content,
		Status:        constant.StatusDraft,
		PublishStatus: pubStatus,
		Type:          articleType,
	}
	id, err := dao.Article().Create(ctx.Request.Context(), a)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"id": id})
}

// AdminArticleUpdate 管理员：修改文章（含已删除的也可修改）
func AdminArticleUpdate(ctx *gin.Context, articleType int) {
	id, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	_, err := dao.Article().GetByIDIncludeDeleted(ctx.Request.Context(), id)
	if err != nil {
		reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
		return
	}
	var body struct {
		Title         *string   `json:"title"`
		Content       *string   `json:"content"`
		UserID        *uint     `json:"user_id"`   // 可选，0 表示置空
		SchoolID      *uint     `json:"school_id"` // 可选，0 表示置空
		PublishStatus *int16    `json:"publish_status"`
		Status        *int16    `json:"status"` // 1正常 2禁用，管理员可直改
		Images        *[]string `json:"images"` // 图片 URL 列表，用于 OSS 上传后更新
	}
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	updates := make(map[string]interface{})
	if body.Title != nil {
		updates["title"] = *body.Title
	}
	if body.Content != nil {
		updates["content"] = *body.Content
	}
	if body.UserID != nil {
		if *body.UserID > 0 {
			_, err := dao.User().GetByID(ctx.Request.Context(), *body.UserID)
			if err != nil {
				reply.ReplyErrWithMessage(ctx, "用户不存在")
				return
			}
			updates["user_id"] = int(*body.UserID)
		} else {
			updates["user_id"] = nil
		}
	}
	if body.SchoolID != nil {
		if *body.SchoolID > 0 {
			_, err := dao.School().GetByID(ctx.Request.Context(), *body.SchoolID)
			if err != nil {
				reply.ReplyErrWithMessage(ctx, "学校不存在")
				return
			}
			updates["school_id"] = int(*body.SchoolID)
		} else {
			updates["school_id"] = 0 // 0 表示全站公开
		}
	}
	if body.PublishStatus != nil {
		updates["publish_status"] = *body.PublishStatus
	}
	if body.Status != nil && (*body.Status == constant.StatusValid || *body.Status == constant.StatusInvalid) {
		updates["status"] = *body.Status
	}
	if body.Images != nil {
		paths := make([]string, len(*body.Images))
		for i, p := range *body.Images {
			paths[i] = oss.PathForStorage(p)
		}
		updates["images"] = pq.StringArray(paths)
		updates["image_count"] = len(paths)
	}
	if len(updates) == 0 {
		reply.ReplyOK(ctx)
		return
	}
	if err := dao.Article().UpdateColumns(ctx.Request.Context(), id, updates); err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// AdminArticleList 管理员：文章列表（帖子/提问/回答）。默认含已删除，可传 include_invalid=0 只看正常
func AdminArticleList(ctx *gin.Context, articleType int) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	schoolID, _ := strconv.ParseUint(ctx.DefaultQuery("school_id", "0"), 10, 32)
	includeInvalid := ctx.Query("include_invalid") != "0" // 默认 true，管理端需看到已删除并可恢复
	list, total, err := dao.Article().ListAdmin(ctx.Request.Context(), uint(schoolID), articleType, includeInvalid, page, pageSize)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	// 将图片路径转为完整 URL 供前端使用
	for _, a := range list {
		a.Images = oss.TransformImageURLs(a.Images)
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": list, "total": total, "page": page, "page_size": pageSize})
}

// AdminSchoolList 管理员：学校列表。默认含已下架，可传 include_invalid=0 只看正常
func AdminSchoolList(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	includeInvalid := ctx.Query("include_invalid") != "0" // 默认 true
	list, total, err := dao.School().List(ctx.Request.Context(), page, pageSize, includeInvalid)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": list, "total": total, "page": page, "page_size": pageSize})
}

// AdminSchoolUpdate 管理员：修改学校信息
func AdminSchoolUpdate(ctx *gin.Context) {
	id, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	if _, err := dao.School().GetByID(ctx.Request.Context(), id); err != nil {
		reply.ReplyNotFound(ctx, errcode.ErrSchoolNotFound)
		return
	}
	var body struct {
		Name       *string        `json:"name"`
		LoginURL   *string        `json:"login_url"`
		Code       *string        `json:"code"`
		FormFields []string       `json:"form_fields"`
		CaptchaURL *string        `json:"captcha_url"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	updates := make(map[string]interface{})
	if body.Name != nil {
		updates["name"] = *body.Name
	}
	if body.LoginURL != nil {
		updates["login_url"] = *body.LoginURL
	}
	if body.Code != nil {
		updates["code"] = *body.Code
	}
	if body.FormFields != nil {
		updates["form_fields"] = body.FormFields
	}
	if body.CaptchaURL != nil {
		updates["captcha_url"] = *body.CaptchaURL
	}
	if len(updates) > 0 {
		if err := dao.School().UpdateColumns(ctx.Request.Context(), id, updates); err != nil {
			reply.ReplyInternalError(ctx, err)
			return
		}
	}
	reply.ReplyOK(ctx)
}

// AdminSchoolCreate 管理员：新增学校
func AdminSchoolCreate(ctx *gin.Context) {
	var body struct {
		Name       string   `json:"name"`
		LoginURL   string   `json:"login_url"`
		Code       string   `json:"code"`
		FormFields []string `json:"form_fields"`
		CaptchaURL string   `json:"captcha_url"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	formFields := body.FormFields
	if len(formFields) == 0 {
		formFields = []string{"username", "password"}
	}
	school := &model.School{
		Name:       strPtr(body.Name),
		LoginURL:   strPtr(body.LoginURL),
		Code:       strPtr(body.Code),
		FormFields: model.FormFieldsJSON(formFields),
		Status:     constant.StatusValid,
	}
	if body.CaptchaURL != "" {
		school.CaptchaURL = &body.CaptchaURL
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
