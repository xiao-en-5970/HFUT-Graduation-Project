-- 订单记录买方选用的收货地址（user_locations.id），便于追溯
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS receiver_user_location_id BIGINT REFERENCES user_locations (id) ON DELETE SET NULL;

COMMENT ON COLUMN orders.receiver_user_location_id IS '下单时选用的买方收货地址 user_locations.id';

CREATE INDEX IF NOT EXISTS idx_orders_receiver_user_location_id ON orders (receiver_user_location_id);
