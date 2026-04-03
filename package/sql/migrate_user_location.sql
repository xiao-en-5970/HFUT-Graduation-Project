-- 用户收货地址（多地址、常用默认、软删除）
CREATE TABLE IF NOT EXISTS user_locations (
                                              id          BIGSERIAL PRIMARY KEY,
                                              user_id     INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
                                              label       VARCHAR(64)  NOT NULL DEFAULT '',
                                              addr        VARCHAR(512) NOT NULL DEFAULT '',
                                              lat         DOUBLE PRECISION,
                                              lng         DOUBLE PRECISION,
                                              is_default  BOOLEAN      NOT NULL DEFAULT FALSE,
                                              status      SMALLINT     NOT NULL DEFAULT 1,
                                              created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
                                              updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE user_locations IS '用户收货地址；status 1 正常 2 软删除';
COMMENT ON COLUMN user_locations.label IS '地址标签（如寝室、家）';
COMMENT ON COLUMN user_locations.addr IS '文字地址';
COMMENT ON COLUMN user_locations.lat IS '地图选点纬度 WGS84，与 lng 成对';
COMMENT ON COLUMN user_locations.lng IS '地图选点经度 WGS84';
COMMENT ON COLUMN user_locations.is_default IS '是否默认收货地址，每用户至多一条为 true（仅 status=1）';
COMMENT ON COLUMN user_locations.status IS '1 正常 2 已删除（软删除）';

CREATE INDEX IF NOT EXISTS idx_user_locations_user_id ON user_locations (user_id);
CREATE INDEX IF NOT EXISTS idx_user_locations_user_status ON user_locations (user_id, status);

-- 每用户至多一条「默认」且未删除
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_locations_one_default
    ON user_locations (user_id)
    WHERE is_default = TRUE AND status = 1;
