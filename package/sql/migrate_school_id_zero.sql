-- 插入 schools id=0 占位行，供 users.school_id=0 表示「未绑定」使用（FK 约束需要）
INSERT INTO schools (id, name) VALUES (0, '未绑定') ON CONFLICT (id) DO NOTHING;
SELECT setval(pg_get_serial_sequence('schools', 'id'), (SELECT COALESCE(MAX(id), 1) FROM schools));

-- 将已有用户的 school_id NULL 统一为 0
UPDATE users SET school_id = 0 WHERE school_id IS NULL;
