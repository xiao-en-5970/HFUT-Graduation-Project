-- 商品统一地址：发货地/自提点，与 pickup_addr 同步使用
ALTER TABLE goods
    ADD COLUMN IF NOT EXISTS goods_addr VARCHAR(512);
COMMENT ON COLUMN goods.goods_addr IS '商品地址：默认卖方发货地与自提约定地址，可与 pickup_addr 同步';

UPDATE goods
SET goods_addr = pickup_addr
WHERE (goods_addr IS NULL OR TRIM(goods_addr) = '')
  AND pickup_addr IS NOT NULL
  AND TRIM(pickup_addr) <> '';
