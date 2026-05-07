-- 商品：可砍价（与「面议」正交；有标价也可刀）
BEGIN;

ALTER TABLE goods
    ADD COLUMN IF NOT EXISTS bargain BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN goods.bargain IS '可刀：接受议价/砍价，与 negotiable(面议) 不同';

COMMIT;
