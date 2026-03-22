-- 旧枚举 1～5 → 新枚举 1～6（仅执行一次；新库无历史数据可跳过）
-- 旧：1 待下单 2 正在派送 3 待买方确认 4 已完成 5 已取消
-- 新：1 待买方付款下单 2 待卖方确认收款 3 履约中 4 待买方确认 5 已完成 6 已取消
-- 自高向低更新，避免中间状态冲突

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
