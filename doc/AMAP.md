# 地图与订单距离（自托管）

已改为 **Martin 矢量瓦片** + **MapLibre** 浏览器选点；订单 **`distance_meters`** 由服务端按两端经纬度计算 **Haversine
球面直线距离**，**不再使用高德**。

## 坐标系

订单中的 `receiver_lat` / `receiver_lng`、`sender_lat` / `sender_lng` 均为 **WGS84**，与 OSM / Martin 一致。

## 环境变量

```bash
# Martin：仅本 Go 进程访问（内网），勿暴露给浏览器
MAP_TILES_URL=http://127.0.0.1:50001/tiles
```

## API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/config/map` | JWT；返回 `map_tiles_url` 为本 API 的瓦片模板（绝对 URL） |
| GET | `/api/v1/map/tiles/:z/:x/:y` | JWT；反向代理至 Martin；MapLibre 请求须带 Bearer |

## 管理后台

- 侧边栏 **地图选点**：管理员登录后使用 MapLibre 点选坐标（与交易演示独立）。
- **交易演示**：买卖双方 JWT；地图以 **浏览器定位** 为中心，**详细位置**手填文字。

## 部署参考

瓦片服务搭建见独立仓库 `map-project`（Martin、`tiles.mbtiles`、`anhui-*.osm.pbf` 等）。
