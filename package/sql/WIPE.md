# wipe_business_data.sql — 上线前删档

清空所有用户产生的业务数据（帖子 / 商品 / 订单 / 通知 / 推荐画像 / 运维指标 / 审计 …），
保留：

- `schools` 整张表
- `users` 中 `role IN (2,3)` 的管理员 / 超级管理员
- `users` 中 `username = '__order_official__'` 的系统订单官方账号

执行后所有业务表自增 ID 复位到 1（next val = 1），后续注册的第一个用户 ID 从「保留账号的 max(id) + 1」起算。

---

## 跑之前一定要备份

```bash
pg_dump "postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME" \
    --no-owner --no-acl > backup_$(date +%Y%m%d_%H%M%S).sql
```

---

## 执行方式 1：宿主机直连

```bash
psql "postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME" \
     -v ON_ERROR_STOP=1 \
     -f package/sql/wipe_business_data.sql
```

`-v ON_ERROR_STOP=1` 让任何一步报错就立刻退出，事务自动回滚，库不留半截脏数据。

---

## 执行方式 2：docker compose 里的 postgres

```bash
cat package/sql/wipe_business_data.sql | \
    docker exec -i hfut-postgres \
        psql -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1
```

把 `hfut-postgres` 换成实际容器名（`docker ps` 查）。

---

## 验证脚本正常

脚本末尾会打印一张表，预期：

| table_name          | rows                             |
|---------------------|----------------------------------|
| users_keep          | 管理员数 + 1（含 `__order_official__`） |
| schools_keep        | 当前已配置的学校数（>=1）                   |
| articles            | 0                                |
| comments            | 0                                |
| likes               | 0                                |
| goods               | 0                                |
| tags                | 0                                |
| collect             | 0                                |
| collect_item        | 0                                |
| follow              | 0                                |
| orders              | 0                                |
| order_messages      | 0                                |
| order_message_reads | 0                                |
| notifications       | 0                                |
| user_cert           | 0                                |
| user_behaviors      | 0                                |
| user_locations      | 0                                |
| metric_minute       | 0                                |
| bot_dispatch_event  | 0                                |
| service_token_audit | 0                                |

如果出现非 0 行（除 users_keep / schools_keep 外），说明有未列入 TRUNCATE 的子表或种子数据被 CASCADE 错过——回滚检查后再跑。

---

## 提示

- 业务后端 / QQ-bot **跑着的状态下也能执行**：事务里第一步 `LOCK TABLE users IN ACCESS EXCLUSIVE MODE`
  会等到现存连接的写事务结束，然后独占。但建议演练前提前停服跑 wipe，避免影响连接重试。
- 跑完 wipe 后**重启 QQ-bot 进程**，因为 bot 进程内有 metrics 累计计数器（重启清零），跟新库对齐更整齐。
- 学校 ID（schools.id）不变；如果之前测试创建了不该留的学校，请手工 `DELETE FROM schools WHERE id = X;` 删除（注意会引发
  user.school_id FK 反查 — wipe 之后 users 里通常没引用了，可以删）。
