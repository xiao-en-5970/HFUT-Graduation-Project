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

// SchoolListForBind 获取可绑定学校列表（含 form_fields、captcha_url）
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
		fields := s.FormFields
		if len(fields) == 0 {
			fields = []model.FormFieldItem{
				{Key: "username", LabelZh: "学号", LabelEn: "Student ID"},
				{Key: "password", LabelZh: "密码", LabelEn: "Password"},
			}
		}
		item["form_fields"] = fields
		if s.CaptchaURL != nil && *s.CaptchaURL != "" {
			item["captcha_url"] = s.CaptchaURL
		} else {
			item["captcha_url"] = nil
		}
		out = append(out, item)
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": out, "total": total})
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
	img, token, err := schools.GetCaptcha(ctx.Request.Context(), *school.Code)
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
