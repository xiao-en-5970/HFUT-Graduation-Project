-- 商品表增加 like_count、collect_count
ALTER TABLE goods
    ADD COLUMN IF NOT EXISTS like_count integer NOT NULL DEFAULT 0;
ALTER TABLE goods
    ADD COLUMN IF NOT EXISTS collect_count integer NOT NULL DEFAULT 0;
COMMENT ON COLUMN goods.like_count IS '点赞次数';
COMMENT ON COLUMN goods.collect_count IS '收藏次数';

-- 订单表增加收货地址、发货地址
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS receiver_addr VARCHAR(512);
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS sender_addr VARCHAR(512);
COMMENT ON COLUMN orders.receiver_addr IS '收货地址';
COMMENT ON COLUMN orders.sender_addr IS '发货地址';

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS distance_meters integer;
COMMENT ON COLUMN orders.distance_meters IS '发货地与收货地步行规划距离（米），高德地图 API 计算';
