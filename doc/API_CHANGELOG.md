# API 变更记录

每次接口文档更新时，在此记录变更内容，供前端同步适配。

格式：按日期倒序，每条列出 **日期**、**变更类型**（新增/修改/废弃）、** affected 接口**、**说明**。

---

## 2026-04-03（下单必选地址簿；订单 receiver_user_location_id）

- **Breaking：** `POST /api/v1/orders` 必填 **`user_location_id`**（买方 `user_locations` 有效记录）；不再接受直接传
  `receiver_addr` / `receiver_lat` / `receiver_lng` 作为下单主路径。
- 订单表新增 **`receiver_user_location_id`**（迁移 `package/sql/migrate_order_receiver_user_location.sql`），详情接口返回该字段。
- 管理端 `POST /api/v1/admin/user-locations` 为指定用户新增地址；后台「收货地址」页提供 **新增地址** 按钮。

---

## 2026-04-03（用户收货地址 user_locations）

- 新增表 **`user_locations`**：多地址、`is_default` 默认、`status` 1 正常 2 软删除；迁移脚本 `package/sql/migrate_user_location.sql`。
- 接口（需 JWT，须放在 `/user/:id` 之前已满足）：`GET/POST /api/v1/user/locations`，`PUT/DELETE /api/v1/user/locations/:id`，`POST /api/v1/user/locations/:id/default`。
- 管理端：`GET /api/v1/admin/user-locations`（`user_id`、`all_status` 筛选分页）、`DELETE /api/v1/admin/user-locations/:id` 软删除；后台侧栏 **收货地址**。

---

## 2026-04-03（订单：自提也算步行距离）

- `POST /api/v1/orders`：商品类型为**自提**时，与送货上门相同，在收发两端均有成对经纬度且配置 `GRAPHHOPPER_BASE_URL` 时写入 `distance_meters`（自提点→买方）。
- 下单时从商品复制 `goods_lat`/`goods_lng` 到订单 `sender_lat`/`sender_lng` 时改为**数值拷贝**，避免指针与查询缓冲共用导致坐标未落库。

---

## 2026-04-03（帖子列表：按更新时间排序）

- 下列列表接口增加查询参数 **`sort`**：传 **`updated_at`** 时按 **`updated_at` 降序**；缺省按 **`created_at` 降序**。
- 涉及：`GET /api/v1/post`、`/post/search`、`/question`、`/question/search`、`/answer`、`/answer/search`、`GET /api/v1/search/articles`（聚合搜索）、`GET /api/v1/user/{id}/posts|questions|answers`。

---

## 2026-04-03（地图瓦片经 API 反向代理）

- `GET /api/v1/config/map` 返回的 `map_tiles_url` 指向本服务 `.../api/v1/map/tiles/{z}/{x}/{y}`；`MAP_TILES_URL` 仅为服务端访问 Martin 的上游地址。
- 新增 `GET /api/v1/map/tiles/:z/:x/:y`（JWT），MapLibre 拉瓦片需 `Authorization: Bearer`。

---

## 2026-04-03（地图：自托管 Martin + GraphHopper，移除高德）

### 废弃 / 移除

| 接口 | 说明 |
|------|------|
| `GET /api/v1/map/district`、`/map/input-tips`、`/map/place-text` | 已删除；不再提供行政区级联与高德 POI 搜索 |

### 修改

| 接口 | 说明 |
|------|------|
| `GET /api/v1/config/map` | 返回 `map_tiles_url`（Martin 模板）；环境变量 `MAP_TILES_URL` |
| `POST/PUT` 订单算距 | 送货上门时仅当**收发均有经纬度**时由服务端调 GraphHopper `foot` 路径距离；`GRAPHHOPPER_BASE_URL` |

坐标：**WGS84**。详见 `doc/AMAP.md`。

### 管理后台

- 「交易演示」：浏览器定位 + MapLibre 点选；详细地址手填字符串。

---

## 2026-04-03（订单地址：文字 + 地图坐标）

### 新增

| 接口                       | 说明                                                          |
|--------------------------|-------------------------------------------------------------|
| `GET /api/v1/config/map` | 返回 `map_tiles_url`（需 JWT）；`MAP_TILES_URL`。 |

### 修改

| 接口                       | 变更说明                                                                                              |
|--------------------------|---------------------------------------------------------------------------------------------------|
| `POST /api/v1/orders`    | 可选 `receiver_lat/lng`、`sender_lat/lng`（WGS84，成对）。送货上门算距：仅**两端均有坐标**时 GraphHopper 步行路网。 |
| `PUT /api/v1/orders/:id` | 卖方可传成对 `sender_lat`/`sender_lng`；与 `sender_addr` 可组合。                                             |

### 数据库

- 执行 `package/sql/migrate_order_addr_coords.sql` 增加 `receiver_lat/lng`、`sender_lat/lng`。

---

## 2026-04-03（订单状态简化与接口文档补全）

### 修改（Breaking）

| 接口                                         | 变更说明                                                                                                                                                                 |
|--------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `POST /api/v1/orders`                      | **请求体**：`goods_id`、`user_location_id`（必填，买方地址簿）、可选 `sender_*`。**不再**接受 `buyer_claim_paid` 等字段。创建成功后 **order_status 恒为 1**，`buyer_agreed_at` 为买方下单时间。详见上方「下单必选地址簿」条目。 |
| `POST /api/v1/orders/:id/buyer-claim-paid` | **已删除**（路由不存在，调用将 404）。线下付款不再通过单独接口声明；以卖方 `seller-confirm-payment` 为准。                                                                                               |

### 订单状态 `order_status`（5 档）

| 值 | 含义                      |
|---|-------------------------|
| 1 | 待卖方确认收款（下单默认）           |
| 2 | 履约中（送货上门：正在派送；自提：待买方自提） |
| 3 | 待买方确认收货                 |
| 4 | 已完成（已扣库存）               |
| 5 | 已取消                     |

**约定：** 不设「待买方付款」状态；卖方长期不确认收款即视为未成交（产品外契约）。详见 `doc/ORDER_AND_CHAT.md`。

### 数据库

- 若历史库曾为 **6 档**枚举，需执行 `package/sql/migrate_order_status_to_5.sql`（见脚本内注释）。新库以 `create.sql` 为准。

### 文档

- `doc/openapi.json`：新增 **商品**、**订单** tags；补充 `/goods`、`/orders`、`/user/{id}/goods` 等路径及 `OrderCreateBody`、
  `OrderDetail` 等定义；`info.description` 中注明状态机与废弃接口。

---

## 2025-02-17（验证码改为前端填写）

### 修改

| 接口 | 变更说明 |
|------|----------|
| `GET /api/v1/schools` | **新增**：学校列表，含 form_fields、captcha_url |
| `GET /api/v1/schools/:id/captcha` | **新增**：获取验证码图片（base64）及 token |
| `POST /api/v1/user/bind/school` | 增加 captcha、captcha_token（form_fields 含 captcha 时必填） |
| `POST /api/v1/user/school-login` | 增加 captcha、captcha_token（需验证码的学校必填） |

### 数据库

- schools 表：form_fields（jsonb）、captcha_url（varchar）
- HFUT：`UPDATE schools SET form_fields = '["username","password","captcha"]'::jsonb WHERE code = 'hfut'`

---

## 2025-02-17（学校登录与绑定）

### 新增

| 接口                               | 变更说明                                            |
|----------------------------------|-------------------------------------------------|
| `POST /api/v1/user/school-login` | 学校端登录，仅需 school_code、username、password，对接学校 CAS |
| `user_cert` 表                    | 记录用户在某学校的认证信息，cert_info 为 JSONB 存储学生信息          |

### 修改

| 接口                              | 变更说明                                                                                                        |
|---------------------------------|-------------------------------------------------------------------------------------------------------------|
| `POST /api/v1/user/bind/school` | **Breaking**：请求体改为 `{ school_id, username, password }`，需学校端账号密码验证；成功后写入 user_cert                           |
| `GET /api/v1/search/articles`   | 热度公式配比全部环境变量可配置：SEARCH_WEIGHT_COLLECT/LIKE/VIEW(10/5/1)、SEARCH_INTERACTION_DECAY_DAYS(90)、SEARCH_COMBINED_* |

---

## 2025-02-17

### 修改

| 接口                            | 变更说明                                                      |
|-------------------------------|-----------------------------------------------------------|
| `GET /api/v1/search/articles` | 描述：补充 zhparser 中文智能分词；排序说明改为 zhparser                     |
| `GET /api/v1/post/search`     | 描述：补充 zhparser；`q` 改为可选，空则退化为列表；page/pageSize 默认值         |
| `GET /api/v1/question/search` | 同上                                                        |
| `GET /api/v1/answer/search`   | 同上                                                        |
| `GET /api/v1/post/drafts`     | 描述：补充 data 结构 `{list,total,page,page_size}`，list 含 author |
| `GET /api/v1/question/drafts` | 同上                                                        |
| `GET /api/v1/answer/drafts`   | 同上                                                        |

---

*后续更新请在本文件顶部追加新日期条目。*
