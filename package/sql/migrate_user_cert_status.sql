-- user_cert 表增加 status 列：1正常 2惰性删除
ALTER TABLE user_cert ADD COLUMN IF NOT EXISTS status smallint NOT NULL DEFAULT 1;
COMMENT ON COLUMN user_cert.status IS '1正常 2惰性删除';
