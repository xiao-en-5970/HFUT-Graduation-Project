# 高德地图（订单距离）

订单表字段 `distance_meters` 表示**发货地与收货地之间的步行规划距离（米）**，由后端调用高德 Web 服务 API 自动计算并写入。

## 地址与坐标（订单）

每笔订单的收货、发货各支持两类信息（均可选组合）：

| 字段                                                        | 含义                     |
|-----------------------------------------------------------|------------------------|
| `receiver_addr` / `sender_addr`                           | 文字描述（寝室、楼号等）           |
| `receiver_lat`+`receiver_lng` / `sender_lat`+`sender_lng` | 地图选点（**GCJ-02**，与高德一致） |

**算距优先级（仅送货上门）**：若收发**两端均提供成对经纬度**，则直接用坐标调用「距离测量」，不再依赖文字地理编码；否则在两段*
*文字地址均非空**时走地理编码再测距。

数据库迁移：`package/sql/migrate_order_addr_coords.sql`（已有库执行一次）。

## 配置

在环境变量或 `.env` 中设置：

```bash
AMAP_KEY=你的Web服务Key
# 可选：管理后台「交易演示」内嵌地图选点用（JS API）
AMAP_WEB_KEY=你的JS端Key
```

未配置 `AMAP_KEY` 时：下单与更新订单**不受影响**，`distance_meters` 为空。未配置 `AMAP_WEB_KEY` 时：后台仍可手动填写经纬度，只是不加载地图。

## 后端封装清单（审计）

**结论：所有高德 Web 服务（`restapi.amap.com`）均由服务端使用 `AMAP_KEY` 调用，前端/ App 不持有 Web 服务 Key、不直连 REST。**
封装落在 `package/amap` 与 `app/controller/map_*.go`、`app/service/order.go`。

| 能力            | 前端/客户端调用的入口                                | 服务端封装位置                                                     | 是否向下游暴露 `AMAP_KEY`                               |
|---------------|--------------------------------------------|-------------------------------------------------------------|--------------------------------------------------|
| 订单步行距离、地理编码测距 | `POST /orders`、`PUT /orders/:id`（内部触发）     | `app/service/order.go` → `package/amap/amap.go`             | 否                                                |
| 行政区（省市区）      | `GET /api/v1/map/district`                 | `app/controller/map_search.go` → `package/amap/district.go` | 否                                                |
| 输入提示          | `GET /api/v1/map/input-tips`               | `map_search.go` → `package/amap/place.go`                   | 否                                                |
| POI 关键字搜索     | `GET /api/v1/map/place-text`               | 同上                                                          | 否                                                |
| 浏览器内嵌地图底图     | `GET /api/v1/config/map` 返回 `amap_web_key` | `app/controller/map_config.go`                              | **仅下发 `AMAP_WEB_KEY`（JS 端 Key），与 Web 服务 Key 分离** |

**说明**：管理后台 `loadAmapScript` 仅向 `https://webapi.amap.com/maps` 加载高德 **JS SDK**（浏览器渲染地图的常规方式）；*
*不得**在管理端或 App 内直接请求 `restapi.amap.com` 并拼接 `AMAP_KEY`。若需原生 App 内嵌高德地图，仍使用移动端 **SDK +
控制台单独配置的 Key**，与上述 REST 封装相互独立。

## 接口（均需 JWT；地图类除 `config/map` 外均使用服务端 `AMAP_KEY`）

| 方法  | 路径                       | 说明                                                              |
|-----|--------------------------|-----------------------------------------------------------------|
| GET | `/api/v1/config/map`     | 返回 `amap_web_key`，供 **Web/H5** 加载高德 JS 地图                       |
| GET | `/api/v1/map/district`   | **行政区下级**：`keywords` 默认 `100000`（中国）取省列表；换省/市 adcode 取下一级（级联）   |
| GET | `/api/v1/map/input-tips` | **输入提示**（联想）：`keywords` 必填，`city`（填 adcode 或城市名）、`citylimit` 可选 |
| GET | `/api/v1/map/place-text` | **POI 关键字搜索**：`keywords` 必填，`city`、`page`、`offset` 可选           |

地名搜索返回 `data.list[]`，每项含 `name`、`address`、`lng`、`lat`、`has_coord`（**GCJ-02**，与订单字段一致）。无坐标条目（如部分公交线）
`has_coord` 为 `false`，勿用于下单坐标。

## 管理后台「交易演示」

推荐流程与手机端一致：**先选省 → 市 → 区县**（调用 `/map/district` 级联），再在「搜详细位置」里输入小区/路名等，**搜索候选**（
`/map/input-tips`，并带 `city=当前选中区划 adcode` + `citylimit=1` 缩小范围），点击一条即可填入收货文字与坐标。也可跳过搜索，直接在地图上点选。

## 手机 App（原生）

可以，与是否 Web 无关：

1. **当前位置**：调用系统定位（iOS Core Location / Android FusedLocation 等）得到经纬度。注意：系统多为 **WGS84**，国内地图展示需转为
   **GCJ-02**（可用高德/腾讯定位 SDK 直接拿国测坐标，或自行做坐标转换后再写入 `receiver_lat`/`lng`）。
2. **地图选点**：集成 **高德 Android/iOS SDK**（或 Flutter 插件），用户拖动图钉后取 GCJ-02 坐标，填入下单接口。
3. **搜地名再选点**：两种方式任选：
    - **推荐**：调用本服务 `GET /api/v1/map/input-tips` 或 `place-text`，列表展示后用户选一条，用返回的 `lng`/`lat`（
      `has_coord=true`）填订单；`name`/`address` 可拼进 `receiver_addr`。
    - 或在 App 内嵌高德 SDK 自带 POI 搜索 UI，坐标同样为 GCJ-02。

这样 **Web 服务 Key 只放在服务端**，App 只需用户登录后的 JWT，无需把 `AMAP_KEY` 打进安装包（若仍要在客户端直调高德，需在控制台为
Key 绑定包名，并承担泄露风险）。

## 流程

1. **下单** `POST /orders`：可传文字与/或地图坐标；送货上门且能算距时写入 `distance_meters`。
2. **卖家修改发货地** `PUT /orders/:id`：可传 `sender_addr` 与/或成对 `sender_lat`/`sender_lng`，按规则重算距离。

## 实现说明

- 地理编码、步行距离：`package/amap/amap.go`
- 输入提示、POI 搜索：`package/amap/place.go`
- 行政区：`package/amap/district.go`
- 地址请尽量写完整（省市区+详细地址），有利于解析成功率。

## 参考

- [地理编码 API](https://lbs.amap.com/api/webservice/guide/api/georegeo)
- [距离测量 API](https://lbs.amap.com/api/webservice/guide/api/distance)
- [输入提示 API](https://lbs.amap.com/api/webservice/guide/api/inputtips)
- [关键字搜索 API](https://lbs.amap.com/api/webservice/guide/api/textsearch)
- [行政区域查询 API](https://lbs.amap.com/api/webservice/guide/api/district)
