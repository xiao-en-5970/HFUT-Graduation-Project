-- P3.7：app 内更新——版本元数据表 + apk 文件由 OSS 存。
--
-- 业务流：
--   1. 开发者本地：make build VERSION=x.y.z 打 release apk → 通过 admin 接口上传
--   2. 后端：admin POST /api/v1/admin/app-release —— 把 apk 存进 OSS（七牛或本地 driver）+ 入库
--      然后自动清理：保留同 platform 下 status=1 的最新 2 个 versionCode，多余的删（连 OSS 文件一起）
--   3. 前端：每次启动 GET /api/v1/app/latest-version → 比对本地 versionCode → 弹"更新/下次再说/忽略此版本"
--
-- 字段语义：
--   platform       预留多端：android（目前唯一）/ ios（暂不开放，App Store 审核流程不需要应用内提示）
--   version_name   语义版本号字符串，跟 build.gradle::versionName / package.json.version 一致
--                  形如 "1.2.3" 或 "1.2.3-rc1"——前端展示用
--   version_code   单调递增整数，跟 build.gradle::versionCode 一致；值由 build-apk.sh 自动算
--                  X.Y.Z → X*10000 + Y*100 + Z；前端比较以这个为准（避免字符串比较踩 "1.10" vs "1.9" 坑）
--   apk_url        七牛/本地 OSS 完整 URL；前端拿到后用系统浏览器打开下载安装（不做 app 内静默更新——
--                  Android 静默装需要系统签名应用 / VPN 流量等高权限，不适合学生项目）
--   release_notes  发布说明 markdown / 纯文本；前端展示在弹窗里
--   force_update   true = 强制更新（弹窗只显示"更新"按钮，不能跳过）；用于线上重大 bug 时
--   status         1=valid（前端能看到）、2=disabled（隐藏；OSS 文件保留以防回滚）
--
-- 设计考虑：
--   * uniq (platform, version_code) 防重复入库
--   * idx (platform, status, version_code DESC) 让 LatestRelease 查询走 index-only scan
--
-- 应用：
--   docker exec -i <hfut-postgres 容器名> psql -U postgres -d graduation_project < migrate_app_releases.sql

CREATE TABLE IF NOT EXISTS app_releases
(
    id            BIGSERIAL PRIMARY KEY,
    platform      VARCHAR(16)  NOT NULL DEFAULT 'android',
    version_name  VARCHAR(32)  NOT NULL,
    version_code  INTEGER      NOT NULL,
    apk_url       VARCHAR(512) NOT NULL,
    release_notes TEXT         NOT NULL DEFAULT '',
    force_update  BOOLEAN      NOT NULL DEFAULT FALSE,
    status        SMALLINT     NOT NULL DEFAULT 1,
    created_at    TIMESTAMP    NOT NULL DEFAULT NOW(),
    CONSTRAINT uniq_app_release_platform_versioncode UNIQUE (platform, version_code)
);

CREATE INDEX IF NOT EXISTS idx_app_releases_active
    ON app_releases (platform, status, version_code DESC);

COMMENT ON TABLE app_releases IS 'app 版本元数据；apk 实体存 OSS（最多保留最新两个 valid 版本）';
COMMENT ON COLUMN app_releases.version_code IS '单调递增整数版本号，跟 Android build.gradle versionCode 对齐（X*10000+Y*100+Z）';
COMMENT ON COLUMN app_releases.apk_url IS 'apk 完整 URL（七牛 / 本地 OSS）；前端用系统浏览器打开下载';
COMMENT ON COLUMN app_releases.force_update IS 'true=强制更新，弹窗禁用"下次再说/忽略"按钮';
COMMENT ON COLUMN app_releases.status IS '1=valid 前端可见；2=disabled 隐藏（OSS 文件保留）';
