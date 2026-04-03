-- 可选：历史系统用户 __order_official__（旧订单消息外键）；新逻辑不再写入 msg_type=3
-- 密码为占位 bcrypt；status=1；登录由应用层拒绝，见 userService.Login

INSERT INTO users (username, password, school_id, status, role)
VALUES (
    '__order_official__',
    '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
    NULL,
    1,
    1
)
ON CONFLICT (username) DO NOTHING;

-- 曾用旧脚本插入为 status=2 的库，统一改为正常
UPDATE users
SET status = 1
WHERE username = '__order_official__';

COMMENT ON COLUMN order_messages.msg_type IS '1:文字 2:图片 3:历史保留（列表不返回）';
