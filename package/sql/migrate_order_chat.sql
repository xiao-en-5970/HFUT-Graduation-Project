-- 订单聊天 + 新状态机（已有库执行一次）
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS buyer_agreed_at TIMESTAMP;
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS seller_agreed_at TIMESTAMP;
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS delivery_images VARCHAR(2048)[];
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS buyer_confirm_images VARCHAR(2048)[];
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS completed_at TIMESTAMP;

COMMENT ON COLUMN orders.order_status IS '1:待下单 2:正在派送 3:待买方确认收货 4:已完成 5:已取消（平台不经手资金）';

-- 旧语义迁移：待支付/已支付/已发货/已收货 -> 已完成(4)
UPDATE orders
SET order_status = 4,
    completed_at = COALESCE(completed_at, updated_at)
WHERE order_status IN (2, 3, 4);
-- 旧「已取消」保持 5

CREATE TABLE IF NOT EXISTS order_messages
(
    id         SERIAL PRIMARY KEY,
    order_id   INTEGER  NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    sender_id  INTEGER  NOT NULL REFERENCES users (id),
    msg_type   SMALLINT NOT NULL DEFAULT 1,
    content    TEXT,
    image_url  VARCHAR(1024),
    created_at TIMESTAMP         DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_order_messages_order_id ON order_messages (order_id);
