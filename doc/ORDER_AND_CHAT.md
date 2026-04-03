# 订单与聊天（平台不经手资金）

本系统**不处理任何资金**，买卖双方线下沟通、扫码付款。

**约定**：买方**下单**即进入 **待卖方确认收款**。卖方不确认收款，即视为未成交（契约上可理解为买方未有效付款或交易未成立），*
*不再单独设「待买方付款」状态**。

## 商品类别 `goods_type`（商品表）

| 值 | 含义       | 订单履约                                                                        |
|---|----------|-----------------------------------------------------------------------------|
| 1 | **送货上门** | 商品 **商品地址** 可作默认 **卖方发货地址**。卖方确认收款后 **正在派送** → `confirm-delivery` → 待买方确认收货 |
| 2 | **自提**   | 卖方确认收款后 **待买方自提**；买方 **确认收货** 表示已提货完成                                       |
| 3 | **在线商品** | 卖方确认收款后 **直接进入待买方确认收货**；无 `confirm-delivery`                                |

## 订单状态 `order_status`（5 档）

| 值 | 含义                         |
|---|----------------------------|
| 1 | **待卖方确认收款**（下单后默认）         |
| 2 | **履约中**：送货上门=正在派送；自提=待买方自提 |
| 3 | **待买方确认收货**                |
| 4 | **已完成**（扣库存）               |
| 5 | **已取消**                    |

## 典型 API 顺序

1. `POST /api/v1/orders` — `{ "goods_id", "receiver_addr"?, "receiver_lat/lng"?, "sender_addr"?, "sender_lat/lng"? }` →
   状态 **1**，并记录 **买方下单时间**（`buyer_agreed_at`）。地图坐标为 **WGS84**；送货上门算距需**两端均有坐标**（服务端 GraphHopper）。
2. `GET /api/v1/config/map` — 取 Martin `map_tiles_url`，供 MapLibre（需 JWT，`MAP_TILES_URL`）。详见 `doc/AMAP.md`。
3. `GET/POST /api/v1/orders/:id/messages` — 聊天（未结束前）。`msg_type`：1 文字、2 图片、**3 官方通知**（仅服务端写入）。卖方确认收款、卖方确认送达、买方确认收货时，会向会话插入一条官方文案，买卖双方均可见。需执行 `package/sql/migrate_order_official_message.sql`（或全新库 `create.sql` 已含系统用户 `__order_official__`）。
4. `POST /api/v1/orders/:id/seller-confirm-payment` — 卖方确认收款 → **2**（在线商品 → **3**）
5. 送货上门：`POST .../confirm-delivery` → **3**；自提：无此步
6. `POST .../confirm-receipt` — 买方 → **4**
7. `POST .../cancel` — 状态 **1、2、3** 可取消 → **5**

卖方更新发货地：`PUT /api/v1/orders/:id` 可传 `sender_addr` 与/或成对 `sender_lat`/`sender_lng`。

图片请先走 `POST /api/v1/oss/...` 上传。

## 管理后台

「交易演示」：`/admin/#/trade-demo`（页面**左栏买方、右栏卖方**，中间为共享聊天记录）。数据库迁移：若曾使用 **6 档**状态，执行 `migrate_order_status_to_5.sql` 一次；官方聊天用户见上文 `migrate_order_official_message.sql`。
