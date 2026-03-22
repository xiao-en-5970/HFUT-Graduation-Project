# 高德地图（订单距离）

订单表字段 `distance_meters` 表示**发货地与收货地之间的步行规划距离（米）**，由后端调用高德 Web 服务 API 自动计算并写入。

## 配置

在环境变量或 `.env` 中设置：

```bash
AMAP_KEY=你的Web服务Key
```

未配置时：下单与更新订单**不受影响**，`distance_meters` 为空。

## 流程

1. **下单** `POST /orders`：若同时传入 `sender_addr` 与 `receiver_addr`，且已配置 `AMAP_KEY`，则地理编码两段地址后调用「距离测量」接口，写入
   `distance_meters`。
2. **卖家补全/修改发货地** `PUT /orders/:id`：当请求体中带 `sender_addr` 时，在更新地址后重新计算距离（需已有
   `receiver_addr`）。

## 实现说明

- 封装见 `package/amap/amap.go`：地理编码 `geocode/geo`、步行距离 `distance`（`type=3`）。
- 地址请尽量写完整（省市区+详细地址），有利于解析成功率。

## 参考

- [地理编码 API](https://lbs.amap.com/api/webservice/guide/api/georegeo)
- [距离测量 API](https://lbs.amap.com/api/webservice/guide/api/distance)
