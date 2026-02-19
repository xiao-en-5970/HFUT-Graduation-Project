package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

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
		if info == nil {
			c.Status(http.StatusNotFound)
			return
		}
		reply.ReplyInternalError(c, err)
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
