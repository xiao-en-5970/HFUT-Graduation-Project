ALTER TABLE goods
    ADD COLUMN IF NOT EXISTS goods_type smallint NOT NULL DEFAULT 1;
ALTER TABLE goods
    ADD COLUMN IF NOT EXISTS pickup_addr VARCHAR(512);
COMMENT ON COLUMN goods.goods_type IS '1:送货上门 2:自提 3:在线商品';
COMMENT ON COLUMN goods.pickup_addr IS '自提类：约定提货地点';
