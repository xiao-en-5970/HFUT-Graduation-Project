package oss

import (
	"errors"
	"fmt"
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

// ToFullURL 将存储的路径转为前端可用的完整 URL。
//
// 三类输入：
//
//  1. 七牛完整 URL（如 https://oss.xiaoen.xyz/good/123/img.jpg）
//     → 加 ?imageView2/2/w/{OSSSmallImageSize}/q/75 query 让七牛 CDN 边缘即时缩略
//     → URL 写入 DB 时不带 query；展示时拼出 query；前端拿到的就是缩略图 URL
//
//  2. 老版本本地完整 URL（如 https://api.xxx/api/v1/oss/path.jpg，含 OSSHost 前缀）
//     → 走 urlAppendSmall 把 .small 后缀塞进 path，沿用旧逻辑
//
//  3. 相对路径（DB 里老条目，driver=local 时存的）
//     → 老逻辑：pathForDisplay 加 .small + 拼 OSSHost + /api/v1/oss/ 前缀
func ToFullURL(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		// 七牛域名 → 用 imageView2 query 替代 .small
		if config.QiniuDomain != "" && strings.HasPrefix(path, config.QiniuDomain) {
			return appendQiniuImageView2(path)
		}
		// 老的本地完整 URL（如带 OSSHost 前缀的）→ 沿用 .small 后缀
		return urlAppendSmall(path)
	}
	p := pathForDisplay(path)
	if config.OSSHost != "" {
		return strings.TrimSuffix(config.OSSHost, "/") + "/api/v1/oss/" + strings.TrimPrefix(p, "/")
	}
	return "/api/v1/oss/" + strings.TrimPrefix(p, "/")
}

// appendQiniuImageView2 给七牛 URL 加缩略图处理参数，等价于 .small 在七牛侧的实现。
//
// 参数含义：
//
//	imageView2/2/w/N/q/75   = mode 2（限定宽度等比缩放） + 宽度 N + JPEG 质量 75
//
// 边界：
//   - 非图片扩展名：原样返回（七牛 imageView2 仅对图片生效，对 PDF 等加了也无害但我们手动跳过）
//   - 已经带其它 query：用 & 拼接而不是 ?
//   - OSSSmallImageSize <= 0：表示禁用缩略图，原样返回原图 URL
func appendQiniuImageView2(rawURL string) string {
	if config.OSSSmallImageSize <= 0 {
		return rawURL
	}
	// 取 path 部分（去掉 query/fragment）判断扩展名
	pathPart := rawURL
	if i := strings.IndexAny(rawURL, "?#"); i >= 0 {
		pathPart = rawURL[:i]
	}
	pathLow := strings.ToLower(pathPart)
	isImage := false
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp"} {
		if strings.HasSuffix(pathLow, ext) {
			isImage = true
			break
		}
	}
	if !isImage {
		return rawURL
	}
	sep := "?"
	if strings.Contains(rawURL, "?") {
		sep = "&"
	}
	return rawURL + sep + fmt.Sprintf("imageView2/2/w/%d/q/75", config.OSSSmallImageSize)
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

// Save 保存上传文件到指定相对路径。
//
// driver 分流（看 config.OSSDriver）：
//
//	"qiniu"           走七牛云，写入 bucket，返回完整 https URL（不带 query）；
//	                   不生成 .small——七牛 imageView2 在 ToFullURL 阶段加 query 即时缩略
//	"local"（默认）    写本地磁盘，同路径无损覆盖（先写临时文件 + 原子替换）；
//	                   启用 OSSSmallImageSize 时同时生成 .small 缩略版
//
// 注意：driver=qiniu 但 QINIU_* 配置不全时（getQiniuClient 报错），**自动兜底走 local**——
// 这是为了"忘配凭证 / 凭证打错"场景下功能不彻底瘫痪；启动时已经打 warning 了。
func Save(file *multipart.FileHeader, relPath string) (url string, err error) {
	relPath = strings.TrimPrefix(relPath, "/")
	if relPath == "" {
		return "", ErrInvalidPath
	}

	// driver=qiniu：走七牛上传；任何错误都打 log 但兜底回 local，避免功能瘫痪
	if config.OSSDriver == "qiniu" {
		if u, qerr := qiniuSave(file, relPath); qerr == nil {
			return u, nil
		} else {
			fmt.Printf("[oss] qiniu Save 失败 key=%s: %v；兜底走 local\n", relPath, qerr)
		}
	}

	// 走本地磁盘
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
	// 若启用压缩且为图片，生成 .small 版本；压缩失败则删除原图并返回错误，避免无效地址入库
	if config.OSSSmallImageSize > 0 {
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(relPath), "."))
		if imageExts[ext] {
			smallPath := fullPath + image.SmallSuffix
			if err := image.CompressToSmall(fullPath, smallPath, uint(config.OSSSmallImageSize), config.OSSSmallImageKB); err != nil {
				os.Remove(fullPath)
				return "", fmt.Errorf("图片压缩失败: %w", err)
			}
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

// Delete 删除指定相对路径的文件，若为图片则同时删除其 .small 版本（仅 local driver）。
//
// 路径分流：
//   - 完整 http(s) URL：判断是不是七牛域名前缀；是 → 走 qiniuDelete；否 → 不操作（外站 URL 不归我们管）
//   - 相对路径："存量本地数据" 或 "驱动=local 时新写入"，走 os.Remove
//
// driver=qiniu 时新文件的 relPath 形式不会经过这里——业务方把"完整 URL"传过来，
// 我们识别 qiniu 域名后截 key 调 qiniuDelete。
func Delete(relPath string) error {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return ErrInvalidPath
	}

	// 完整 URL：判断是不是七牛域名（DB 里 driver=qiniu 时存的就是完整 URL）
	if strings.HasPrefix(relPath, "http://") || strings.HasPrefix(relPath, "https://") {
		if config.QiniuDomain != "" && strings.HasPrefix(relPath, config.QiniuDomain) {
			// 截掉 domain + "/" 拿 key
			key := strings.TrimPrefix(relPath, config.QiniuDomain)
			key = strings.TrimPrefix(key, "/")
			// 去掉 ?imageView2/... 这类 query
			if i := strings.IndexAny(key, "?#"); i >= 0 {
				key = key[:i]
			}
			return qiniuDelete(key)
		}
		// 外站 URL（如果有的话）—— 不删除
		return nil
	}

	// 相对路径：本地磁盘
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

// ArticleImagePath 帖子图片存储路径 article/{articleId}/image_{index}.{ext}（已弃用，新上传请用 ArticleImagePathWithSnowflake）
func ArticleImagePath(articleID uint, index int, ext string) string {
	ext = strings.TrimPrefix(ext, ".")
	if ext == "" {
		ext = "jpg"
	}
	return "article/" + strconv.FormatUint(uint64(articleID), 10) + "/image_" + strconv.Itoa(index) + "." + ext
}

// ArticleImagePathWithSnowflake 帖子图片存储路径（雪花 ID 避免冲突）article/{articleId}/img_{snowflake}.{ext}
func ArticleImagePathWithSnowflake(articleID uint, snowflakeID int64, ext string) string {
	ext = strings.TrimPrefix(ext, ".")
	if ext == "" {
		ext = "jpg"
	}
	return "article/" + strconv.FormatUint(uint64(articleID), 10) + "/img_" + strconv.FormatInt(snowflakeID, 10) + "." + ext
}

// GoodImagePathWithSnowflake 商品图片存储路径 good/{goodId}/img_{snowflake}.{ext}
func GoodImagePathWithSnowflake(goodID uint, snowflakeID int64, ext string) string {
	ext = strings.TrimPrefix(ext, ".")
	if ext == "" {
		ext = "jpg"
	}
	return "good/" + strconv.FormatUint(uint64(goodID), 10) + "/img_" + strconv.FormatInt(snowflakeID, 10) + "." + ext
}

// BotUserImagePath bot 转存的图片存储路径 user/{userId}/bot/img_{snowflake}.{ext}
//
// 跟 web 端"先建 good 再上传图"的链路区别：bot 那边一次性识别完商品就调 PublishGood，
// good_id 还没生出来——所以用 user/ 前缀的"用户级临时图床"路径，跟具体 good/article 解耦。
//
// 这条路径下面的图：
//   - 落库到 goods.images / articles.images 后会被前端正常引用
//   - good 被下架/删除时不连带删图（路径下没有 good_id 关联，做不到自动级联）——
//     长期会堆积，P3 阶段加 cleanup 任务（按"该用户在 7 天前的 bot/img_*"做清理）
func BotUserImagePath(userID uint, snowflakeID int64, ext string) string {
	ext = strings.TrimPrefix(ext, ".")
	if ext == "" {
		ext = "jpg"
	}
	return "user/" + strconv.FormatUint(uint64(userID), 10) + "/bot/img_" + strconv.FormatInt(snowflakeID, 10) + "." + ext
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

// ExtractUserAssetPath 从 URL 或路径中提取 user/{id}/ 形式的存储路径；校验是否为指定用户的头像/背景。
//
// 支持的输入格式（按出现顺序匹配，先命中先用）：
//   - 七牛完整 URL：https://oss.xiaoen.xyz/user/123/avatar.jpg[?imageView2/...]
//   - 老本地完整 URL：https://api.xxx.com/api/v1/oss/user/123/avatar.jpg
//   - 本地 API 路径：/api/v1/oss/user/123/avatar.jpg
//   - 直接相对路径：user/123/avatar.jpg
//
// 切到七牛 driver 后，前端拿到的 URL 是七牛域名形式；用户在 PUT 个人信息时把这个
// URL 回传给后端，必须能正常识别到 user/{id}/ 路径——否则会误判"路径无效或无权使用"。
func ExtractUserAssetPath(input string, userID uint) (storagePath string, err error) {
	if input == "" {
		return "", ErrInvalidPath
	}
	p := strings.TrimSpace(input)

	// 1) 七牛域名前缀 → 直接 trim 出 key
	if config.QiniuDomain != "" && strings.HasPrefix(p, config.QiniuDomain) {
		p = strings.TrimPrefix(p, config.QiniuDomain)
		p = strings.TrimPrefix(p, "/")
	} else {
		// 2) 兼容老格式：含 /api/v1/oss/ 前缀（无论前面是否有 host）
		if idx := strings.Index(p, "/api/v1/oss/"); idx >= 0 {
			p = strings.TrimPrefix(p[idx:], "/api/v1/oss/")
		} else {
			p = strings.TrimPrefix(p, "/")
			p = strings.TrimPrefix(p, "api/v1/oss/")
		}
		// 还残留 http(s):// 说明既不是七牛、也不含本地 API 路径——拒收
		if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
			return "", ErrInvalidPath
		}
	}

	// 去掉 query string（七牛 imageView2 / 老 .small 等）
	if idx := strings.Index(p, "?"); idx >= 0 {
		p = p[:idx]
	}
	// 兼容：老本地路径可能带 .small 后缀，这里统一剥掉
	p = strings.TrimSuffix(p, image.SmallSuffix)

	if !strings.HasPrefix(p, "user/") {
		return "", ErrInvalidPath
	}
	// 期望格式 user/{id}/avatar.* 或 user/{id}/background.*
	parts := strings.SplitN(p, "/", 3)
	if len(parts) < 3 {
		return "", ErrInvalidPath
	}
	idStr := parts[1]
	id, e := strconv.ParseUint(idStr, 10, 32)
	if e != nil || uint(id) != userID {
		return "", ErrInvalidPath
	}
	// 禁止目录穿越
	if strings.Contains(p, "..") {
		return "", ErrInvalidPath
	}
	return p, nil
}
