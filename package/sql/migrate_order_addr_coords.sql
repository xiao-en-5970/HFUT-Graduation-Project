-- 订单收货/发货：文字地址 + 地图选点坐标（GCJ-02），便于步行距离计算
-- 仅执行一次

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS receiver_lat DOUBLE PRECISION;
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS receiver_lng DOUBLE PRECISION;
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS sender_lat DOUBLE PRECISION;
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS sender_lng DOUBLE PRECISION;

COMMENT ON COLUMN orders.receiver_addr IS '收货地址文字说明（寝室/楼号等口头描述）';
COMMENT ON COLUMN orders.receiver_lat IS '收货地图选点纬度 GCJ-02，与 receiver_lng 成对使用';
COMMENT ON COLUMN orders.receiver_lng IS '收货地图选点经度 GCJ-02';
COMMENT ON COLUMN orders.sender_addr IS '发货地址文字说明';
COMMENT ON COLUMN orders.sender_lat IS '发货地图选点纬度 GCJ-02';
COMMENT ON COLUMN orders.sender_lng IS '发货地图选点经度 GCJ-02';
