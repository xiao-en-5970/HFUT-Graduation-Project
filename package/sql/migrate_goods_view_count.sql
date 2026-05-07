-- 给 goods 表加 view_count —— 商品详情页浏览次数。
--
-- 动机：
--   原本只有 articles 表有 view_count，goods 表没有，前端"X 浏览"对商品永远显示 0。
--   推荐侧的 GoodRecommendParams.PopularityView 也只是占位，实际未参与打分。
--
-- 实现：
--   - service.goodService.Get（GET /goods/:id 入口）成功通过可见性校验且
--     status=valid + good_status=在售(1) 时，dao.Good().IncrViewCount(id) 原子 +1
--   - 自己看自己 +1（与 articles 一致；不做"作者豁免"以免反直觉）
--   - 短时间多次刷新仍 +1；想精细去重可在网关 / Redis 层加 (user, good, day) 缓存
--
-- 应用：
--   docker exec -i <hfut-postgres 容器名> psql -U postgres -d graduation_project < migrate_goods_view_count.sql

ALTER TABLE goods
    ADD COLUMN IF NOT EXISTS view_count integer NOT NULL DEFAULT 0;

COMMENT ON COLUMN goods.view_count IS '浏览次数：商品详情页每次成功访问 +1（仅 status=valid + good_status=1 在售商品计入）';
