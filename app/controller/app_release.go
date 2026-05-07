package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// AppLatestVersionResp 公开接口返回体——刻意只暴露前端弹窗需要的字段，
// 不带 id / status / created_at 等内部状态。
type AppLatestVersionResp struct {
	Platform     string `json:"platform"`
	VersionName  string `json:"version_name"`
	VersionCode  int    `json:"version_code"`
	APKURL       string `json:"apk_url"`
	ReleaseNotes string `json:"release_notes"`
	ForceUpdate  bool   `json:"force_update"`
}

// AppLatestVersion GET /api/v1/app/latest-version?platform=android
//
// 公开接口（无需 JWT），前端启动时拉来跟本地 versionCode 对比；
// 暂时无版本时返 200 + null（不返 404，让前端能区分"接口异常"和"暂无版本"）。
//
// 设计原因：404 跟"网络抖+服务挂"都是非 200，前端要分别处理麻烦；
// 200+null 是常见的"找不到也不算错"语义，前端只需 if (data) 判断即可。
func AppLatestVersion(c *gin.Context) {
	platform := c.DefaultQuery("platform", service.AppPlatformAndroid)
	rec, err := service.AppRelease().LatestValidPublic(c.Request.Context(), platform)
	if err != nil {
		reply.ReplyErrWithMessage(c, err.Error())
		return
	}
	if rec == nil {
		reply.ReplyOKWithData(c, nil)
		return
	}
	resp := buildLatestResp(rec)
	reply.ReplyOKWithData(c, resp)
}

func buildLatestResp(rec *model.AppRelease) *AppLatestVersionResp {
	// APK URL 透传——七牛 driver 时已是完整公网 https URL；
	// local driver 时需走 ToFullURL（拼上 OSSHost）让前端能直接 GET。
	url := rec.APKURL
	if url != "" {
		url = oss.ToFullURL(url)
	}
	return &AppLatestVersionResp{
		Platform:     rec.Platform,
		VersionName:  rec.VersionName,
		VersionCode:  rec.VersionCode,
		APKURL:       url,
		ReleaseNotes: rec.ReleaseNotes,
		ForceUpdate:  rec.ForceUpdate,
	}
}

// AdminAppReleaseUpload POST /api/v1/admin/app-release —— admin 上传新版本 apk。
//
// multipart/form-data：
//   - file          apk 文件（必填）
//   - version_name  语义版本号字符串（必填）
//   - version_code  整数版本号（必填，需跟 build.gradle::versionCode 一致）
//   - release_notes 发布说明（可选）
//   - force_update  true/false（可选；默认 false）
//   - platform      android（可选；默认 android）
//
// 上传成功后会异步清理超过 keepLatestN 的旧版本（连 OSS 文件一起删）。
func AdminAppReleaseUpload(c *gin.Context) {
	var req service.CreateReleaseReq
	if err := c.ShouldBind(&req); err != nil {
		reply.ReplyInvalidParams(c, err)
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		reply.ReplyErrWithMessage(c, "缺少 apk 文件（form 字段名 file）")
		return
	}
	rec, err := service.AppRelease().Create(c, req, file)
	if err != nil {
		reply.ReplyErrWithMessage(c, err.Error())
		return
	}
	reply.ReplyOKWithData(c, gin.H{
		"id":           rec.ID,
		"platform":     rec.Platform,
		"version_name": rec.VersionName,
		"version_code": rec.VersionCode,
		"apk_url":      oss.ToFullURL(rec.APKURL),
	})
}

// AdminAppReleaseDelete DELETE /api/v1/admin/app-release/:id —— admin 物理删除某条版本（连 OSS）。
//
// 幂等：找不到也返 OK。
func AdminAppReleaseDelete(c *gin.Context) {
	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id64 == 0 {
		reply.ReplyErrWithMessage(c, "无效的 id")
		return
	}
	if err := service.AppRelease().Delete(c.Request.Context(), uint(id64)); err != nil {
		reply.ReplyErrWithMessage(c, err.Error())
		return
	}
	reply.ReplyOK(c)
}

// AdminAppReleaseList GET /api/v1/admin/app-release?platform=android&page=1&page_size=20
//
// 列出所有 status 的版本（含已 disabled 的），admin 用来排查 / 回滚。
func AdminAppReleaseList(c *gin.Context) {
	platform := c.DefaultQuery("platform", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	list, total, err := service.AppRelease().List(c.Request.Context(), platform, page, pageSize)
	if err != nil {
		reply.ReplyErrWithMessage(c, err.Error())
		return
	}
	rows := make([]*AppLatestVersionResp, 0, len(list))
	for _, r := range list {
		rows = append(rows, buildLatestResp(r))
	}
	reply.ReplyOKWithData(c, gin.H{
		"total": total,
		"list":  rows,
	})
}
