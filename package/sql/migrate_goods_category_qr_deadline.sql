-- 2026-05-02 商品：新增类别/收款码/截止时间
-- goods_category : 1 二手买卖（默认）/ 2 有偿求助
-- payment_qr_url : 卖家收款码图片（仅二手买卖有意义，可空）
-- has_deadline   : 是否设置定时下架
-- deadline       : 到期自动下架的时间（UTC 时间戳），当 has_deadline=false 时忽略
-- 定时任务每 5 分钟扫描一次 good_status=1 AND has_deadline=true AND deadline<NOW() 的记录

BEGIN;

ALTER TABLE goods
    ADD COLUMN IF NOT EXISTS goods_category SMALLINT     NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS payment_qr_url VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS has_deadline   BOOLEAN      NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS deadline       TIMESTAMPTZ;

COMMENT ON COLUMN goods.goods_category IS '商品类别 1二手买卖 2有偿求助';
COMMENT ON COLUMN goods.payment_qr_url IS '卖家收款码图片路径（oss path），仅二手买卖使用';
COMMENT ON COLUMN goods.has_deadline IS '是否启用定时下架';
COMMENT ON COLUMN goods.deadline IS '定时下架时间点；仅 has_deadline=TRUE 时生效';

-- 专门给 cron 扫描用的窄索引：只覆盖还在售且启用了截止时间的记录
CREATE INDEX IF NOT EXISTS idx_goods_auto_offshelf
    ON goods (deadline)
    WHERE good_status = 1 AND has_deadline = TRUE;

COMMIT;
