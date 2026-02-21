package controller

import (
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// ossCheckPathPermission 校验 OSS 路径写权限：user/{id}/ 仅本人，article/{id}/ 仅文章所有者，school/{id}/ 仅管理员
func ossCheckPathPermission(c *gin.Context, path string) (allowed bool, msg string) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return false, "未认证"
	}
	parts := strings.SplitN(strings.Trim(path, "/"), "/", 3)
	if len(parts) < 2 {
		return false, "路径格式无效"
	}
	prefix, idStr := parts[0], parts[1]
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return false, "路径 ID 无效"
	}
	switch prefix {
	case "user":
		if uint(id) != userID {
			return false, "无权操作他人文件"
		}
		return true, ""
	case "article":
		ok, e := dao.Article().IsOwnedByUserForOSS(c.Request.Context(), uint(id), userID)
		if e != nil || !ok {
			return false, "无权操作该文章的文件"
		}
		return true, ""
	case "school":
		user, e := dao.User().GetByID(c.Request.Context(), userID)
		if e != nil || user == nil || user.Role < constant.RoleAdmin {
			return false, "仅管理员可操作学校文件"
		}
		return true, ""
	default:
		return false, "不支持的路径类型"
	}
}

// OSSGet 下载/访问文件 GET /api/v1/oss/*path
func OSSGet(c *gin.Context) {
	path := c.Param("path")
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		reply.ReplyErrWithMessage(c, "路径无效")
		return
	}
	info, fullPath, err := oss.StatOrEnsureSmall(path)
	if err != nil {
		if errors.Is(err, oss.ErrInvalidPath) {
			reply.ReplyErrWithMessage(c, "路径非法")
			return
		}
		if info == nil && !os.IsNotExist(err) {
			// 非「文件不存在」的异常（如压缩失败）返回 500
			reply.ReplyInternalError(c, err)
			return
		}
		c.Status(http.StatusNotFound)
		return
	}
	if info.IsDir() {
		reply.ReplyErrWithMessage(c, "不支持目录访问")
		return
	}
	c.File(fullPath)
}

// OSSUpload 通用上传 POST /api/v1/oss/*path
func OSSUpload(c *gin.Context) {
	path := c.Param("path")
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		reply.ReplyErrWithMessage(c, "路径无效")
		return
	}
	if ok, msg := ossCheckPathPermission(c, path); !ok {
		reply.ReplyErrWithMessage(c, msg)
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		reply.ReplyInvalidParams(c, err)
		return
	}
	if strings.HasSuffix(path, "/") {
		path = path + file.Filename
	}
	url, err := oss.Save(file, path)
	if err != nil {
		reply.ReplyInternalError(c, err)
		return
	}
	reply.ReplyOKWithData(c, gin.H{"url": url, "path": path})
}

// OSSDelete 通用删除 DELETE /api/v1/oss/*path
func OSSDelete(c *gin.Context) {
	path := c.Param("path")
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		reply.ReplyErrWithMessage(c, "路径无效")
		return
	}
	if ok, msg := ossCheckPathPermission(c, path); !ok {
		reply.ReplyErrWithMessage(c, msg)
		return
	}
	if err := oss.Delete(path); err != nil {
		if errors.Is(err, oss.ErrInvalidPath) {
			reply.ReplyErrWithMessage(c, "路径非法")
			return
		}
		reply.ReplyInternalError(c, err)
		return
	}
	reply.ReplyOK(c)
}
