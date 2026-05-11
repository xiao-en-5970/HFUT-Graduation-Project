-- 修复 users.avatar / users.background 末尾被错误写入的 `.small` 后缀。
--
-- 背景：早期 `oss.PathForStorage` 实现等价于 `pathForDisplay`，对 oss.Save 返回的
-- 裸图片 URL（七牛模式下没 query 没 .small 后缀）会**主动加 .small**——
-- 导致 DB 里写成了 `https://oss.xiaoen.xyz/.../avatar.jpg.small` 这种七牛上不存在的 key，
-- 前端读取时 ToFullURL 直接返回该 URL 给前端 → 404。
--
-- 头像/背景受影响最严重——它们是 service.UploadAvatar / UploadBackground 直接拿
-- oss.Save 返回的裸 URL 喂给 PathForStorage 的链路（不像商品/文章图片是经过"前端展示
-- URL → 用户回传"循环、URL 末尾自带 query 而绕过 pathForDisplay 的副作用）。
--
-- 修复后 `PathForStorage` 已经改成"真正清洁存储路径"——下次写入不会再加 .small；
-- 这条 migration 一次性把已有脏数据清理掉。读取时 ToFullURL 会按当前 driver 自动
-- 重新拼出合适的 query/.small 后缀。
--
-- 应用：
--   docker exec -i <hfut-postgres> psql -U postgres -d graduation_project < migrate_fix_avatar_small_suffix.sql

UPDATE users
SET avatar = regexp_replace(avatar, '\.small$', '')
WHERE avatar LIKE '%.small';

UPDATE users
SET background = regexp_replace(background, '\.small$', '')
WHERE background LIKE '%.small';

-- 同时清理一下可能带 query 的脏数据（避免历史写入累积导致 DB 字段长度溢出）。
-- 形如 `https://oss.xiaoen.xyz/user/1/avatar.jpg?imageView2/2/w/720/q/75`——读取时
-- ToFullURL 会自动再加 query，这里只是确保 DB 里存的是 raw URL。
UPDATE users
SET avatar = split_part(avatar, '?', 1)
WHERE avatar LIKE '%?%';

UPDATE users
SET background = split_part(background, '?', 1)
WHERE background LIKE '%?%';
