package service

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/logger"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"go.uber.org/zap"
)

// AppReleaseService app 内更新功能的 service。
//
// 业务流详见 package/sql/migrate_app_releases.sql 文件头注释。
type appReleaseService struct{}

// AppRelease 工厂入口，参考其他 service 的命名风格（service.AppRelease().XXX(...)）。
func AppRelease() *appReleaseService { return &appReleaseService{} }

// AppReleaseValid 平台合法值；目前只放 android。新增端时这里加白名单 + 加 migration。
const AppPlatformAndroid = "android"

// keepLatestN 保留最新几个 valid 版本——多出来的 admin 上传后会被自动清理（连 OSS 文件）。
//
// 选 2 是个 trade-off：
//   - 太小（=1）：用户没及时更新就拿不到任何旧版下载链接，热修发布前的旧 apk 也访问不到
//   - 太大：占空间；APK 一般 30-80MB，3+ 个意义不大
const keepLatestN = 2

// LatestValidPublic 返回该平台最新 valid 版本，**屏蔽内部细节后**给前端用。
//
// 找不到时返 (nil, nil)——没发布过任何版本是合理状态，不算错。
func (s *appReleaseService) LatestValidPublic(ctx context.Context, platform string) (*model.AppRelease, error) {
	platform = strings.TrimSpace(platform)
	if platform == "" {
		platform = AppPlatformAndroid
	}
	if platform != AppPlatformAndroid {
		return nil, errors.New("不支持的 platform，目前仅支持 android")
	}
	return dao.AppRelease().LatestValid(ctx, platform)
}

// CreateReleaseReq admin 上传 apk 入参。
//
// VersionName 必填，前端展示用；语义版本号，比如 "1.2.3"
// VersionCode 必填，单调递增整数；前端比对用，跟 build.gradle versionCode 对齐
// ReleaseNotes 选填，发布说明
// ForceUpdate 默认 false；true 时前端弹窗禁用"下次再说/忽略"按钮
type CreateReleaseReq struct {
	Platform     string `form:"platform"`
	VersionName  string `form:"version_name"  binding:"required"`
	VersionCode  int    `form:"version_code"  binding:"required"`
	ReleaseNotes string `form:"release_notes"`
	ForceUpdate  bool   `form:"force_update"`
}

// Create admin 上传新 apk —— 落 OSS、入库、自动清理超过 keepLatestN 的旧版本。
//
// 返回 created record；失败时已上传到 OSS 的 apk 会尝试 best-effort 删除。
func (s *appReleaseService) Create(
	ctx *gin.Context,
	req CreateReleaseReq,
	file *multipart.FileHeader,
) (*model.AppRelease, error) {
	platform := strings.TrimSpace(req.Platform)
	if platform == "" {
		platform = AppPlatformAndroid
	}
	if platform != AppPlatformAndroid {
		return nil, errors.New("不支持的 platform，目前仅支持 android")
	}
	if req.VersionCode <= 0 {
		return nil, errors.New("version_code 必须 > 0")
	}
	if file == nil {
		return nil, errors.New("apk 文件必传")
	}
	// 后端不强校验 .apk 后缀（七牛 / local 都按二进制存）；但加日志便于排查
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".apk") {
		logger.Warn(ctx.Request.Context(), "AppRelease.Create 上传文件名不是 .apk 后缀",
			zap.String("filename", file.Filename))
	}

	// 上传 OSS：路径 release/<platform>/<version_code>-<timestamp>.apk
	// 加 timestamp 是为了同 version_code 重传（比如发布后发现签名错了想覆盖）也能区分；
	// 同 (platform, version_code) 入库时 uniq 索引兜底——不会真有两条 valid 记录指向同一个 versionCode
	relPath := fmt.Sprintf("release/%s/%d-%d.apk",
		platform, req.VersionCode, time.Now().Unix())
	url, err := oss.Save(file, relPath)
	if err != nil {
		return nil, fmt.Errorf("上传 apk 失败: %w", err)
	}

	rec := &model.AppRelease{
		Platform:     platform,
		VersionName:  strings.TrimSpace(req.VersionName),
		VersionCode:  req.VersionCode,
		APKURL:       url,
		ReleaseNotes: strings.TrimSpace(req.ReleaseNotes),
		ForceUpdate:  req.ForceUpdate,
		Status:       constant.StatusValid,
	}
	if err := dao.AppRelease().Create(ctx.Request.Context(), rec); err != nil {
		// uniq 冲突 / DB 错——把刚上传的 OSS 文件回滚删掉，避免孤儿
		if delErr := oss.Delete(url); delErr != nil {
			logger.Warn(ctx.Request.Context(), "AppRelease.Create 回滚 OSS 文件失败",
				zap.String("url", url), zap.Error(delErr))
		}
		// uniq 冲突给个人话错误
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("version_code=%d 已存在；改 status / 删除后再试", req.VersionCode)
		}
		return nil, fmt.Errorf("写入版本记录失败: %w", err)
	}

	// 异步清理超过 keepLatestN 的旧版本——不阻塞 admin 响应；
	// 失败仅 log（不影响发布主流程，下次清理还会再尝试）
	go func() {
		bg, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.pruneOlderVersions(bg, platform); err != nil {
			logger.Warn(bg, "AppRelease.pruneOlderVersions 失败",
				zap.String("platform", platform), zap.Error(err))
		}
	}()

	return rec, nil
}

// Delete admin 物理删除某条版本——OSS 文件一并删除。
//
// 找不到时不报错（幂等）；OSS 删失败仅 log，DB 仍然删（避免"删不掉的脏数据"卡住流程）。
func (s *appReleaseService) Delete(ctx context.Context, id uint) error {
	rec, err := dao.AppRelease().GetByID(ctx, id)
	if err != nil {
		return err
	}
	if rec == nil {
		return nil
	}
	if rec.APKURL != "" {
		if delErr := oss.Delete(rec.APKURL); delErr != nil {
			logger.Warn(ctx, "AppRelease.Delete OSS 文件失败",
				zap.Uint("id", id), zap.String("url", rec.APKURL), zap.Error(delErr))
		}
	}
	return dao.AppRelease().HardDelete(ctx, id)
}

// List admin 端分页列表（任意 status）。
func (s *appReleaseService) List(ctx context.Context, platform string, page, pageSize int) ([]*model.AppRelease, int64, error) {
	return dao.AppRelease().ListAll(ctx, platform, page, pageSize)
}

// pruneOlderVersions 把同平台 valid 记录按 version_code DESC 列出，超过 keepLatestN 的尾部全删。
//
// 删除策略：物理删 + OSS 文件一并删。被删的版本前端不再可见、也不再占 OSS 空间。
// 想要"软删除保留 OSS 文件"应该走 admin 改 status=2，那是另外一条手动路径。
func (s *appReleaseService) pruneOlderVersions(ctx context.Context, platform string) error {
	list, err := dao.AppRelease().ListValid(ctx, platform)
	if err != nil {
		return err
	}
	if len(list) <= keepLatestN {
		return nil
	}
	for i := keepLatestN; i < len(list); i++ {
		old := list[i]
		if err := s.Delete(ctx, old.ID); err != nil {
			logger.Warn(ctx, "AppRelease.prune 单条删除失败（跳过继续）",
				zap.Uint("id", old.ID), zap.Int("version_code", old.VersionCode), zap.Error(err))
			continue
		}
		logger.Info(ctx, "AppRelease.prune 已清理旧版本",
			zap.Uint("id", old.ID), zap.Int("version_code", old.VersionCode))
	}
	return nil
}

// isUniqueViolation 检测是不是 Postgres uniq 索引冲突（错误码 23505）。
//
// gorm 不暴露强类型错误，靠字符串匹配——这是常见做法，跟 hfut 其他地方一致。
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "duplicate key") ||
		strings.Contains(s, "23505") ||
		strings.Contains(s, "UNIQUE constraint")
}
