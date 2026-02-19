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
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/image"
)

var imageExts = map[string]bool{
	"jpg": true, "jpeg": true, "png": true, "gif": true, "webp": true,
}

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

// GetRelPath 获取相对路径对应的 API 路径。若配置了 OSS_HOST 则返回完整 URL。图片默认带 .small 后缀返回压缩版
func GetRelPath(relPath string) string {
	p := pathForDisplay(relPath)
	path := "/api/v1/oss/" + strings.TrimPrefix(p, "/")
	if config.OSSHost != "" {
		return strings.TrimSuffix(config.OSSHost, "/") + path
	}
	return path
}

// pathForDisplay 若为图片且启用了压缩，则路径加 .small，供前端默认加载压缩图
func pathForDisplay(path string) string {
	if path == "" || config.OSSSmallImageSize <= 0 {
		return path
	}
	if strings.HasSuffix(path, image.SmallSuffix) {
		return path
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	if imageExts[ext] {
		return path + image.SmallSuffix
	}
	return path
}

// ToFullURL 将存储的路径转为前端可用的完整 URL。图片默认返回带 .small 的压缩版。若已是完整 URL 则在其路径加 .small
func ToFullURL(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		// 完整 URL：在路径末尾（不含 query）加 .small
		return urlAppendSmall(path)
	}
	p := pathForDisplay(path)
	if config.OSSHost != "" {
		return strings.TrimSuffix(config.OSSHost, "/") + "/api/v1/oss/" + strings.TrimPrefix(p, "/")
	}
	return "/api/v1/oss/" + strings.TrimPrefix(p, "/")
}

func urlAppendSmall(rawURL string) string {
	if config.OSSSmallImageSize <= 0 || strings.Contains(rawURL, image.SmallSuffix) {
		return rawURL
	}
	pathPart := rawURL
	if i := strings.IndexAny(rawURL, "?#"); i >= 0 {
		pathPart = rawURL[:i]
	}
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp"} {
		if strings.HasSuffix(pathPart, ext) {
			return pathPart + image.SmallSuffix + rawURL[len(pathPart):]
		}
	}
	return rawURL
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
	// 若启用压缩且为图片，生成 .small 版本
	if config.OSSSmallImageSize > 0 {
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(relPath), "."))
		if imageExts[ext] {
			smallPath := fullPath + image.SmallSuffix
			_ = image.CompressToSmall(fullPath, smallPath, uint(config.OSSSmallImageSize), config.OSSSmallImageKB)
		}
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

// Delete 删除指定相对路径的文件，若为图片则同时删除其 .small 版本
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
	if err := os.Remove(fullPath); err != nil {
		return err
	}
	// 若为图片，删除 .small 版本
	if !strings.HasSuffix(relPath, image.SmallSuffix) {
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(relPath), "."))
		if imageExts[ext] {
			os.Remove(fullPath + image.SmallSuffix)
		}
	}
	return nil
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

// StatOrEnsureSmall 获取文件，若请求 .small 且不存在则从原图压缩生成后返回（兜底逻辑）
func StatOrEnsureSmall(relPath string) (os.FileInfo, string, error) {
	info, fullPath, err := Stat(relPath)
	if err == nil {
		return info, fullPath, err
	}
	if !os.IsNotExist(err) {
		return info, fullPath, err
	}
	// .small 文件不存在，尝试从原图生成
	if !strings.HasSuffix(relPath, image.SmallSuffix) {
		return info, fullPath, err
	}
	origPath := image.StripSmallSuffix(relPath)
	origInfo, origFullPath, origErr := Stat(origPath)
	if origErr != nil || origInfo == nil || origInfo.IsDir() {
		return info, fullPath, err
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(origPath), "."))
	if !imageExts[ext] {
		return info, fullPath, err
	}
	maxPx := config.OSSSmallImageSize
	if maxPx <= 0 {
		maxPx = 720
	}
	maxKB := config.OSSSmallImageKB
	if maxKB <= 0 {
		maxKB = 200
	}
	if compErr := image.CompressToSmall(origFullPath, fullPath, uint(maxPx), maxKB); compErr != nil {
		return nil, fullPath, compErr
	}
	return Stat(relPath)
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

// PathForStorage 统一存 .small（缩略图），供数据库存储；若已是 .small 或非图片则原样返回
func PathForStorage(path string) string {
	return pathForDisplay(path)
}

// ExtFromFilename 从文件名获取扩展名，如 "x.jpg" -> "jpg"
func ExtFromFilename(filename string) string {
	idx := strings.LastIndex(filename, ".")
	if idx < 0 {
		return "jpg"
	}
	return filename[idx+1:]
}
