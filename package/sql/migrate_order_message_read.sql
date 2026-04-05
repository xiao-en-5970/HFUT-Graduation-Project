-- 订单聊天已读游标：未读 = 对方发送且 id > last_read_message_id
CREATE TABLE IF NOT EXISTS order_message_reads
(
    id                   BIGSERIAL PRIMARY KEY,
    user_id              BIGINT      NOT NULL,
    order_id             BIGINT      NOT NULL,
    last_read_message_id BIGINT      NOT NULL DEFAULT 0,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uniq_order_message_read_user_order UNIQUE (user_id, order_id)
);

CREATE INDEX IF NOT EXISTS idx_order_message_reads_user_id ON order_message_reads (user_id);

COMMENT ON TABLE order_message_reads IS '用户对订单会话的最后已读消息 id，用于未读计数';
