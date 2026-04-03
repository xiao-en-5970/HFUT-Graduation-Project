# API 变更记录

每次接口文档更新时，在此记录变更内容，供前端同步适配。

格式：按日期倒序，每条列出 **日期**、**变更类型**（新增/修改/废弃）、** affected 接口**、**说明**。

---

## 2026-04-03（地图：省市区级联 + 搜详细位置）

### 新增

| 接口                           | 说明                                              |
|------------------------------|-------------------------------------------------|
| `GET /api/v1/map/district`   | 高德行政区下级，省→市→区级联（`keywords` 用 adcode，如 `100000`） |
| `GET /api/v1/map/input-tips` | 高德输入提示，`keywords` 必填                            |
| `GET /api/v1/map/place-text` | 高德 POI 关键字搜索                                    |

均需 JWT；使用服务端 `AMAP_KEY`。手机 App 与 `doc/AMAP.md` 说明一致。

### 管理后台

- 「交易演示」：先选省/市/区县，再「搜详细位置」；候选点选填入收货文字与坐标（管理员登录态调接口）。

---

## 2026-04-03（订单地址：文字 + 地图坐标）

### 新增

| 接口                       | 说明                                                          |
|--------------------------|-------------------------------------------------------------|
| `GET /api/v1/config/map` | 返回 `amap_web_key`、`amap_security_js_code`（需 JWT）；环境变量 `AMAP_WEB_KEY`、`AMAP_WEB_SECURITY_CODE`。 |

### 修改

| 接口                       | 变更说明                                                                                              |
|--------------------------|---------------------------------------------------------------------------------------------------|
| `POST /api/v1/orders`    | 可选 `receiver_lat/lng`、`sender_lat/lng`（GCJ-02，成对）。送货上门算距：**两端均有坐标**时按坐标测距，否则两段**文字地址均非空**时地理编码测距。 |
| `PUT /api/v1/orders/:id` | 卖方可传成对 `sender_lat`/`sender_lng`；与 `sender_addr` 可组合。                                             |

### 数据库

- 执行 `package/sql/migrate_order_addr_coords.sql` 增加 `receiver_lat/lng`、`sender_lat/lng`。

---

## 2026-04-03（订单状态简化与接口文档补全）

### 修改（Breaking）

| 接口                                         | 变更说明                                                                                                                                                   |
|--------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------|
| `POST /api/v1/orders`                      | **请求体**：仅 `goods_id`（必填）、`receiver_addr`、`sender_addr`。**不再**接受 `buyer_claim_paid` 等字段。创建成功后 **order_status 恒为 1**（待卖方确认收款），`buyer_agreed_at` 为买方下单时间。 |
| `POST /api/v1/orders/:id/buyer-claim-paid` | **已删除**（路由不存在，调用将 404）。线下付款不再通过单独接口声明；以卖方 `seller-confirm-payment` 为准。                                                                                 |

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
