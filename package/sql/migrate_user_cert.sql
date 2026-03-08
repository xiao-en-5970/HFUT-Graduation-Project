-- 学校表增加 code 字段，用于对接 package/schools 登录模块（如 hfut）
ALTER TABLE schools
    ADD COLUMN IF NOT EXISTS code VARCHAR(32) UNIQUE;
COMMENT ON COLUMN schools.code IS '学校代码，如 hfut，用于 school-login';

-- 若已有合肥工业大学记录，可执行： UPDATE schools SET code = 'hfut' WHERE name LIKE '%合肥工业%' AND (code IS NULL OR code = '');

-- 用户认证表：记录用户在某学校的认证信息
CREATE TABLE IF NOT EXISTS user_cert
(
    id         SERIAL PRIMARY KEY,
    user_id    integer NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    school_id  integer NOT NULL REFERENCES schools (id) ON DELETE CASCADE,
    cert_info  jsonb   NOT NULL DEFAULT '{}',
    created_at TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, school_id)
);

COMMENT ON TABLE user_cert IS '用户学校认证记录';
COMMENT ON COLUMN user_cert.cert_info IS '学生信息 JSON，来自学校端接口响应';
