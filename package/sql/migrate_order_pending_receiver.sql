-- 买方申请修改收货地址：待卖方确认前暂存
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS pending_receiver_user_location_id INTEGER;
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS pending_receiver_addr VARCHAR(512) DEFAULT '';
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS pending_receiver_lat DOUBLE PRECISION;
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS pending_receiver_lng DOUBLE PRECISION;

COMMENT ON COLUMN orders.pending_receiver_user_location_id IS '买方申请修改的地址簿 id，卖方确认后写入 receiver_user_location_id';
COMMENT ON COLUMN orders.pending_receiver_addr IS '买方申请修改的收货地址快照';
COMMENT ON COLUMN orders.pending_receiver_lat IS '买方申请修改的收货纬度';
COMMENT ON COLUMN orders.pending_receiver_lng IS '买方申请修改的收货经度';
