package controller

import (
	"encoding/base64"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/schools"
)

// SchoolListForBind 获取可绑定学校列表（不含 form_fields，仅 id/name/code）
// GET /api/v1/schools
func SchoolListForBind(ctx *gin.Context) {
	list, total, err := dao.School().List(ctx.Request.Context(), 1, 100, false)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	var out []map[string]interface{}
	for _, s := range list {
		if s.Status != constant.StatusValid || s.Code == nil || *s.Code == "" {
			continue
		}
		item := map[string]interface{}{
			"id":   s.ID,
			"name": s.Name,
			"code": s.Code,
		}
		out = append(out, item)
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": out, "total": total})
}

// SchoolDetailForBind 获取学校详情（含 form_fields、captcha_url、login_url，用于绑定表单）
// GET /api/v1/schools/:id
func SchoolDetailForBind(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		reply.ReplyErrWithMessage(ctx, "学校ID无效")
		return
	}
	school, err := dao.School().GetByID(ctx.Request.Context(), uint(id))
	if err != nil {
		reply.ReplyNotFound(ctx, errcode.ErrSchoolNotFound)
		return
	}
	if school.Status != constant.StatusValid || school.Code == nil || *school.Code == "" {
		reply.ReplyErrWithMessage(ctx, "该学校暂不支持")
		return
	}
	fields := school.FormFields
	if len(fields) == 0 {
		fields = []model.FormFieldItem{
			{Key: "username", LabelZh: "学号", LabelEn: "Student ID"},
			{Key: "password", LabelZh: "密码", LabelEn: "Password"},
		}
	}
	item := map[string]interface{}{
		"id":          school.ID,
		"name":        school.Name,
		"code":        school.Code,
		"form_fields": fields,
	}
	if school.CaptchaURL != nil && *school.CaptchaURL != "" {
		item["captcha_url"] = *school.CaptchaURL
	} else {
		item["captcha_url"] = nil
	}
	if school.LoginURL != nil && *school.LoginURL != "" {
		item["login_url"] = *school.LoginURL
	} else {
		item["login_url"] = nil
	}
	reply.ReplyOKWithData(ctx, item)
}

// SchoolCaptcha 获取学校验证码图片及 token
// GET /api/v1/schools/:id/captcha
func SchoolCaptcha(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		reply.ReplyErrWithMessage(ctx, "学校ID无效")
		return
	}
	school, err := dao.School().GetByID(ctx.Request.Context(), uint(id))
	if err != nil {
		reply.ReplyNotFound(ctx, errcode.ErrSchoolNotFound)
		return
	}
	if school.Code == nil || *school.Code == "" {
		reply.ReplyErrWithMessage(ctx, "该学校暂不支持")
		return
	}
	if school.FormFields.HasKey("captcha") {
		if school.CaptchaURL == nil || *school.CaptchaURL == "" {
			reply.ReplyErrWithMessage(ctx, "该学校未配置验证码地址(captcha_url)，请先在学校管理中配置")
			return
		}
		if school.LoginURL == nil || *school.LoginURL == "" {
			reply.ReplyErrWithMessage(ctx, "该学校未配置登录地址(login_url)，请先在学校管理中配置")
			return
		}
	}
	opts := &schools.CaptchaOptions{
		LoginURL:   "",
		CaptchaURL: "",
	}
	if school.LoginURL != nil {
		opts.LoginURL = *school.LoginURL
	}
	if school.CaptchaURL != nil {
		opts.CaptchaURL = *school.CaptchaURL
	}
	img, token, err := schools.GetCaptcha(ctx.Request.Context(), *school.Code, opts)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	if img == nil || token == "" {
		reply.ReplyErrWithMessage(ctx, "该学校不支持验证码获取")
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{
		"image": base64.StdEncoding.EncodeToString(img),
		"token": token,
	})
}
