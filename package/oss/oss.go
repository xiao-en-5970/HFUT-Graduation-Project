package oss

import (
	"errors"

	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
)

var ErrInvalidPath = errors.New("invalid path")

// SafePath 校验相对路径，防止目录穿越，返回绝对路径
func SafePath(relPath string) (fullPath string, ok bool) {
	relPath = strings.TrimPrefix(relPath, "/")
	relPath = strings.TrimSpace(relPath)
	if relPath == "" || strings.Contains(relPath, "..") {
		return "", false
	}
	// 禁止绝对路径
	if filepath.IsAbs(relPath) {
		return "", false
	}
	full := filepath.Join(config.OSSRoot, relPath)
	abs, err := filepath.Abs(full)
	if err != nil {
		return "", false
	}
	ossAbs, _ := filepath.Abs(config.OSSRoot)
	if !strings.HasPrefix(abs, ossAbs) {
		return "", false
	}
	return abs, true
}

// GetRelPath 获取相对路径对应的 API 路径。若配置了 OSS_HOST 则返回完整 URL 给前端
func GetRelPath(relPath string) string {
	path := "/api/v1/oss/" + strings.TrimPrefix(relPath, "/")
	if config.OSSHost != "" {
		return strings.TrimSuffix(config.OSSHost, "/") + path
	}
	return path
}

// ToFullURL 将存储的路径转为前端可用的完整 URL，存路径取 URL。若已是完整 URL 则原样返回
func ToFullURL(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if config.OSSHost != "" {
		return strings.TrimSuffix(config.OSSHost, "/") + "/" + strings.TrimPrefix(path, "/")
	}
	return path
}

// TransformImageURLs 将图片路径数组转为完整 URL 数组
func TransformImageURLs(urls []string) []string {
	if len(urls) == 0 {
		return urls
	}
	out := make([]string, len(urls))
	for i, p := range urls {
		out[i] = ToFullURL(p)
	}
	return out
}

// Save 保存上传文件到指定相对路径，同路径无损覆盖（先写临时文件，成功后再原子替换）
func Save(file *multipart.FileHeader, relPath string) (url string, err error) {
	relPath = strings.TrimPrefix(relPath, "/")
	if relPath == "" {
		return "", ErrInvalidPath
	}
	fullPath, ok := SafePath(relPath)
	if !ok {
		return "", ErrInvalidPath
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", err
	}
	tmpPath := fullPath + ".tmp"
	if err := saveUploadedFile(file, tmpPath); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	if err := os.Rename(tmpPath, fullPath); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	return GetRelPath(relPath), nil
}

func saveUploadedFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, src)
	return err
}

// Delete 删除指定相对路径的文件
func Delete(relPath string) error {
	fullPath, ok := SafePath(relPath)
	if !ok {
		return ErrInvalidPath
	}
	info, err := os.Stat(fullPath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return ErrInvalidPath
	}
	return os.Remove(fullPath)
}

// Stat 获取文件信息与绝对路径
func Stat(relPath string) (os.FileInfo, string, error) {
	fullPath, ok := SafePath(relPath)
	if !ok {
		return nil, "", ErrInvalidPath
	}
	info, err := os.Stat(fullPath)
	return info, fullPath, err
}

// UserAvatarPath 用户头像存储路径 user/{id}/avatar.{ext}
func UserAvatarPath(userID uint, ext string) string {
	ext = strings.TrimPrefix(ext, ".")
	if ext == "" {
		ext = "jpg"
	}
	return "user/" + strconv.FormatUint(uint64(userID), 10) + "/avatar." + ext
}

// UserBackgroundPath 用户背景图存储路径 user/{id}/background.{ext}
func UserBackgroundPath(userID uint, ext string) string {
	ext = strings.TrimPrefix(ext, ".")
	if ext == "" {
		ext = "jpg"
	}
	return "user/" + strconv.FormatUint(uint64(userID), 10) + "/background." + ext
}

// ArticleImagePath 帖子图片存储路径 article/{articleId}/image_{index}.{ext}
func ArticleImagePath(articleID uint, index int, ext string) string {
	ext = strings.TrimPrefix(ext, ".")
	if ext == "" {
		ext = "jpg"
	}
	return "article/" + strconv.FormatUint(uint64(articleID), 10) + "/image_" + strconv.Itoa(index) + "." + ext
}

// ExtFromFilename 从文件名获取扩展名，如 "x.jpg" -> "jpg"
func ExtFromFilename(filename string) string {
	idx := strings.LastIndex(filename, ".")
	if idx < 0 {
		return "jpg"
	}
	return filename[idx+1:]
}
