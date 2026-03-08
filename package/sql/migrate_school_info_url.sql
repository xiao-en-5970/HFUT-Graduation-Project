-- info 接口配置（禁止写死）：eam_service_url 用于 CAS cookie 换取 EAM session，info_url 为学生信息页 base
ALTER TABLE schools ADD COLUMN IF NOT EXISTS eam_service_url VARCHAR(512);
ALTER TABLE schools ADD COLUMN IF NOT EXISTS info_url VARCHAR(512);
COMMENT ON COLUMN schools.eam_service_url IS 'EAM SSO 地址，CAS cookie 换取 EAM session 用';
COMMENT ON COLUMN schools.info_url IS '学生信息页 base URL，请求 /info/{code} 获取完整信息';
