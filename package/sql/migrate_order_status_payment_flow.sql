-- 【历史】仅当数据库仍为「旧 5 档」且需迁到「中间 6 档」时执行一次。
-- 当前线上目标为 5 档（1 待卖方确认收款 … 5 已取消），若已执行过本脚本，
-- 请继续执行 migrate_order_status_to_5.sql 将 6 档收拢为 5 档。
-- 新库请直接使用 create.sql，勿执行本文件。

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
