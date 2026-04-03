-- 商品发货地地图坐标（与 goods_addr 一致；下单时作为默认发货端坐标）
ALTER TABLE goods
    ADD COLUMN IF NOT EXISTS goods_lat DOUBLE PRECISION;
ALTER TABLE goods
    ADD COLUMN IF NOT EXISTS goods_lng DOUBLE PRECISION;

COMMENT ON COLUMN goods.goods_lat IS '商品位置纬度 WGS84，与发货地一致';
COMMENT ON COLUMN goods.goods_lng IS '商品位置经度 WGS84';
