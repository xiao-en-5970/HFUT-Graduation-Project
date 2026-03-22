-- 迁移：为 likes 表添加联合唯一约束，保证幂等性
-- 若 create.sql 已包含新结构，无需执行
-- 若有重复数据 (user_id, ext_id, ext_type)，需先清理后再执行

CREATE UNIQUE INDEX IF NOT EXISTS uk_likes_user_ext ON likes (user_id, ext_id, ext_type);


ALTER TABLE goods
    ADD COLUMN IF NOT EXISTS goods_addr VARCHAR(512);
COMMENT ON COLUMN goods.goods_addr IS '商品地址：默认卖方发货地与自提约定地址，可与 pickup_addr 同步';

UPDATE goods
SET goods_addr = pickup_addr
WHERE (goods_addr IS NULL OR TRIM(goods_addr) = '')
  AND pickup_addr IS NOT NULL
  AND TRIM(pickup_addr) <> '';

UPDATE orders
SET order_status = 6
WHERE order_status = 5;
UPDATE orders
SET order_status = 5
WHERE order_status = 4;
UPDATE orders
SET order_status = 4
WHERE order_status = 3;
UPDATE orders
SET order_status = 3
WHERE order_status = 2;


UPDATE orders
SET order_status = 5
WHERE order_status = 6;
UPDATE orders
SET order_status = 4
WHERE order_status = 5;
UPDATE orders
SET order_status = 3
WHERE order_status = 4;
UPDATE orders
SET order_status = 2
WHERE order_status = 3;
UPDATE orders
SET order_status = 1
WHERE order_status = 2;