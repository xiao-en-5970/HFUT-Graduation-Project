-- 将 6 档状态压缩为 5 档（去掉「待买方付款下单」；下单即原状态 2）
-- 旧：1 待买方付款 2 待卖方收款 3 履约 4 待确认收货 5 完成 6 取消
-- 新：1 待卖方收款 2 履约 3 待确认收货 4 完成 5 取消
-- 自高向低更新，避免冲突

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
-- 原状态 1（待买方付款）与现状态 1（待卖方收款）同码，无需再改
