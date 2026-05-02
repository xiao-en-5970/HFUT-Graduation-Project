-- 通知聚合：同一作品/评论的点赞在"未读窗口"内聚合为一条；顶层评论和回复不聚合。
--
-- count        : 当前这条通知已累计的触发者数量（含自己以外的去重人数）
-- updated_at   : 最近一次被追加触发者的时间，用于列表按最新活动排序
-- contributors : 已计入本条通知的 from_user_id 列表（用于去重，避免同一人反复点赞被重复计数）
--
-- 聚合只作用于 type=1（赞作品）、type=2（赞评论），且 is_read=false；已读后新点赞会开新一条。
ALTER TABLE notifications
    ADD COLUMN IF NOT EXISTS count        INTEGER   NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS updated_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ADD COLUMN IF NOT EXISTS contributors JSONB     NOT NULL DEFAULT '[]'::jsonb;

-- 历史数据补齐：已有点赞通知按单人计数
UPDATE notifications
SET contributors = jsonb_build_array(from_user_id),
    updated_at   = created_at
WHERE (contributors IS NULL OR contributors = '[]'::jsonb)
  AND type IN (1, 2);

COMMENT ON COLUMN notifications.count IS '已聚合的触发者人数（仅 type=1/2 使用，其他类型固定为 1）';
COMMENT ON COLUMN notifications.updated_at IS '聚合最近一次更新时间，列表按此排序';
COMMENT ON COLUMN notifications.contributors IS '已计入的 from_user_id 列表（JSONB 数组），用于同人去重';

-- 聚合查询专用部分索引：仅命中未读且属于两种点赞类型的行
CREATE INDEX IF NOT EXISTS idx_notifications_aggregate_unread
    ON notifications (user_id, type, target_type, target_id)
    WHERE is_read = FALSE AND status = 1 AND type IN (1, 2);

-- 列表按最新活动排序
CREATE INDEX IF NOT EXISTS idx_notifications_user_updated
    ON notifications (user_id, status, updated_at DESC);
