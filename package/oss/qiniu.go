// Package oss 的 qiniu.go 实现"七牛云 Kodo"作为后端的存储 driver。
//
// 当 config.OSSDriver=="qiniu" 时，oss.Save / oss.Delete 会路由到这里。
// 不是 driver 时这文件里的代码完全不会被调用，七牛 SDK 也是 lazy init（只在
// 第一次走到 qiniu driver 时才初始化全局 client）。
//
// 关键设计：
//
//  1. **key == relPath**：七牛 bucket 里的对象 key 跟原本地相对路径完全一致
//     （好处：迁移工具完全不需要 key 转换；rollback 到 local 时 key 还能照旧用）
//
//  2. **直接写七牛**，不走"先落本地再异步推"——简化失败语义、无并发问题、
//     节省服务器 IO；上传慢由 hfut 自身请求超时兜底。
//
//  3. **不生成 .small**：七牛 imageView2 在边缘 CDN 即时生成缩略图，
//     由 oss.ToFullURL 拼接 URL query 触发；qiniuSave 只存原图。
//
//  4. **返回完整 URL 入库**：DB 里 goods.images / articles.images 等字段
//     直接存 "<QiniuDomain>/<key>"（不带 query）。前端读时再走 ToFullURL
//     加上 imageView2 query。这样 URL 在切回 local driver 后也仍工作。
//
// 依赖：github.com/qiniu/go-sdk/v7
package oss

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"strings"
	"sync"

	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
)

// qiniuMu 保护 qiniuClientCache 的并发初始化。
var qiniuMu sync.Mutex

// qiniuClientCache 全局七牛 client 缓存。
//
// SDK 的 storage.FormUploader / BucketManager 都设计成可以长生命周期复用，内部
// 持有 http.Client 连接池。我们在第一次调用时按 config 字段构造一份，之后所有
// Save / Delete 共用——除非 config 改了（hot reload）才需要重建（现在简化处理：
// 不支持 hot rebuild；config.WatchAndReload 只更新 env，client 会沿用旧 mac+
// region；运维想换七牛凭证就重启 hfut 进程）。
type qiniuClient struct {
	mac        *qbox.Mac
	cfg        *storage.Config
	uploader   *storage.FormUploader
	bucketMgr  *storage.BucketManager
	bucketName string
	domain     string // 完整 URL 前缀，如 "https://oss.xiaoen.xyz"
}

var qiniuClientCache *qiniuClient

// getQiniuClient 拿（或懒构造）全局七牛 client。
//
// 错误：config 不全时返 ErrInvalidPath（语义近似，让上层回 4xx）。
func getQiniuClient() (*qiniuClient, error) {
	qiniuMu.Lock()
	defer qiniuMu.Unlock()

	// 已构造、且 config 没变 → 复用
	if qiniuClientCache != nil &&
		qiniuClientCache.bucketName == config.QiniuBucket &&
		qiniuClientCache.domain == config.QiniuDomain {
		return qiniuClientCache, nil
	}

	// 校验配置
	ak := config.QiniuAccessKey
	sk := config.QiniuSecretKey
	bucket := config.QiniuBucket
	domain := config.QiniuDomain
	regionCode := config.QiniuRegion
	if ak == "" || sk == "" || bucket == "" || domain == "" || regionCode == "" {
		return nil, errors.New("oss/qiniu: QINIU_* 配置不全（AK/SK/BUCKET/DOMAIN/REGION）")
	}

	region, err := resolveQiniuRegion(regionCode)
	if err != nil {
		return nil, err
	}

	mac := qbox.NewMac(ak, sk)
	cfg := &storage.Config{
		Region:        region,
		UseHTTPS:      strings.HasPrefix(domain, "https://"), // 上传也尽量 https，跟下载域名协议一致
		UseCdnDomains: false,                                 // 上传走源站 endpoint，下载走 domain；不需要 CDN 加速上传
	}

	cli := &qiniuClient{
		mac:        mac,
		cfg:        cfg,
		uploader:   storage.NewFormUploader(cfg),
		bucketMgr:  storage.NewBucketManager(mac, cfg),
		bucketName: bucket,
		domain:     domain,
	}
	qiniuClientCache = cli
	return cli, nil
}

// resolveQiniuRegion 把人类可读的区域代号映射成 SDK 的 storage.Region。
//
// 接受新版（cn-south-1 / cn-east-1 / cn-north-1 / cn-east-2 / na0 / as0）
// 和老版别名（z0/z1/z2/cnEast2 等）；不区分大小写、容忍空格。
func resolveQiniuRegion(code string) (*storage.Region, error) {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "cn-east-1", "z0", "huadong":
		return &storage.ZoneHuadong, nil
	case "cn-north-1", "z1", "huabei":
		return &storage.ZoneHuabei, nil
	case "cn-south-1", "z2", "huanan":
		return &storage.ZoneHuanan, nil
	case "cn-east-2", "z3", "cneast2", "huadong2":
		return &storage.ZoneHuadongZheJiang2, nil
	case "na0", "beimei", "north-america":
		return &storage.ZoneBeimei, nil
	case "as0", "xinjiapo", "singapore", "as-east-1":
		return &storage.ZoneXinjiapo, nil
	default:
		return nil, fmt.Errorf("oss/qiniu: 未知 region 代号 %q（支持 cn-east-1/cn-north-1/cn-south-1/cn-east-2/na0/as0 + z0/z1/z2 别名）", code)
	}
}

// qiniuSave 把 multipart 文件上传到七牛 bucket。
//
// key 跟传入的 relPath 完全一致；返回的 URL 是 "<domain>/<key>"（不带 query）。
//
// 失败语义：
//   - 凭证缺失 / region 解析错 → fail fast
//   - SDK 上传错 → fail fast，让 controller 回 500
//   - 文件 size 上限 / 类型限制 不在这层做（保留给 service / controller 层 + 七牛 bucket 配置）
func qiniuSave(file *multipart.FileHeader, relPath string) (string, error) {
	relPath = strings.TrimPrefix(strings.TrimSpace(relPath), "/")
	if relPath == "" {
		return "", ErrInvalidPath
	}
	if strings.Contains(relPath, "..") {
		// 防 key 注入，跟 SafePath 同口径
		return "", ErrInvalidPath
	}

	cli, err := getQiniuClient()
	if err != nil {
		return "", err
	}

	// 上传 token：限定 scope=bucket:key，policy 仅允许这一个 key 覆盖写入；
	// 一次性 token，不缓存（签名几个微秒，无意义优化）。
	policy := storage.PutPolicy{
		Scope: cli.bucketName + ":" + relPath, // bucket:key 表示"覆盖上传到这个 key"
	}
	upToken := policy.UploadToken(cli.mac)

	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("oss/qiniu: 打开 multipart 失败: %w", err)
	}
	defer src.Close()

	var ret storage.PutRet
	putExtra := storage.PutExtra{}
	if err := cli.uploader.Put(context.Background(), &ret, upToken, relPath, src, file.Size, &putExtra); err != nil {
		return "", fmt.Errorf("oss/qiniu: 上传失败 key=%s: %w", relPath, err)
	}

	// 七牛 PutRet 里 ret.Key 跟我们传的 relPath 一致，hash 不入 DB
	return cli.domain + "/" + relPath, nil
}

// qiniuDelete 从七牛 bucket 删除一个 key。
//
// 行为对齐 local Delete：key 不存在不算错（让 retry 友好），其它错原样冒上去。
func qiniuDelete(relPath string) error {
	relPath = strings.TrimPrefix(strings.TrimSpace(relPath), "/")
	if relPath == "" {
		return ErrInvalidPath
	}
	cli, err := getQiniuClient()
	if err != nil {
		return err
	}
	if err := cli.bucketMgr.Delete(cli.bucketName, relPath); err != nil {
		// 七牛 612 = 文件不存在，照本地 Delete 不存在静默接受的语义忽略
		if isQiniuNotFound(err) {
			return nil
		}
		return fmt.Errorf("oss/qiniu: 删除失败 key=%s: %w", relPath, err)
	}
	return nil
}

// isQiniuNotFound 判断错误是不是"文件不存在"。
//
// 七牛 SDK 的错误把 HTTP code 编进 *storage.ErrorInfo；612 = 删除时 key 不存在。
func isQiniuNotFound(err error) bool {
	if err == nil {
		return false
	}
	// SDK 返回 *storage.ErrorInfo；用 type assertion 拿 code
	type httpStatusErr interface {
		HttpCode() int
	}
	if e, ok := err.(httpStatusErr); ok {
		return e.HttpCode() == 612
	}
	// 兜底：错误文字里含 "no such file" / "612"
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such file") || strings.Contains(msg, "612")
}
