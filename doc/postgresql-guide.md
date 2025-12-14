# PostgreSQL 序列（Sequence）概念

## 什么是序列？

序列（Sequence）是 PostgreSQL 中用于生成唯一数字序列的对象，类似于 MySQL 的 `AUTO_INCREMENT`，但更强大和灵活。

### 基本概念

1. **独立对象**：序列是数据库中的独立对象，不依赖于表
2. **可共享**：多个表可以共享同一个序列
3. **可控制**：可以手动设置起始值、步长、最大值等
4. **事务安全**：序列操作是事务安全的

### 创建和使用序列

```sql
-- 方式1: 显式创建序列
CREATE SEQUENCE user_id_seq START 1 INCREMENT 1;

-- 在表中使用序列
CREATE TABLE users (
    id BIGINT PRIMARY KEY DEFAULT nextval('user_id_seq'),
    name VARCHAR(100)
);

-- 方式2: 使用 SERIAL 类型（自动创建序列）
CREATE TABLE users (
    id SERIAL PRIMARY KEY,  -- 自动创建 users_id_seq 序列
    name VARCHAR(100)
);

-- 方式3: 使用 BIGSERIAL（大整数序列）
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100)
);
```

### 序列操作

```sql
-- 获取下一个值
SELECT nextval('user_id_seq');

-- 获取当前值（不增加）
SELECT currval('user_id_seq');

-- 设置序列值
SELECT setval('user_id_seq', 100);

-- 查看序列信息
SELECT * FROM user_id_seq;

-- 删除序列
DROP SEQUENCE user_id_seq;
```

### 在 GORM 中使用

GORM 会自动处理 PostgreSQL 的序列，使用 `SERIAL` 或 `BIGSERIAL` 类型即可：

```go
type User struct {
    ID   uint   `gorm:"primaryKey"`
    Name string
}
// GORM 会自动使用 SERIAL 类型
```

---

# MySQL 到 PostgreSQL 迁移指南

## 1. 数据类型差异

### 字符串类型
- **MySQL**: `VARCHAR(n)`, `TEXT`
- **PostgreSQL**: `VARCHAR(n)`, `TEXT`（基本相同）
- **注意**: PostgreSQL 的 `TEXT` 没有长度限制，性能与 `VARCHAR` 相同

### 整数类型
- **MySQL**: `INT`, `BIGINT`
- **PostgreSQL**: `INTEGER`, `BIGINT`（基本相同）
- **注意**: PostgreSQL 没有 `TINYINT`，使用 `SMALLINT` 代替

### 自增主键
- **MySQL**: `AUTO_INCREMENT`
- **PostgreSQL**: `SERIAL`, `BIGSERIAL`（使用序列实现）

```sql
-- MySQL
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY
);

-- PostgreSQL
CREATE TABLE users (
    id SERIAL PRIMARY KEY
);
```

### 布尔类型
- **MySQL**: `TINYINT(1)` 或 `BOOLEAN`
- **PostgreSQL**: `BOOLEAN`（原生支持，推荐使用）

### 日期时间
- **MySQL**: `DATETIME`, `TIMESTAMP`
- **PostgreSQL**: `TIMESTAMP`, `TIMESTAMPTZ`（带时区）
- **注意**: PostgreSQL 推荐使用 `TIMESTAMPTZ` 存储带时区的时间

### JSON 类型
- **MySQL**: `JSON`（5.7+）
- **PostgreSQL**: `JSON`, `JSONB`（推荐 `JSONB`，支持索引）

## 2. SQL 语法差异

### 字符串拼接
```sql
-- MySQL
SELECT CONCAT(first_name, ' ', last_name) AS full_name;

-- PostgreSQL
SELECT first_name || ' ' || last_name AS full_name;
-- 或
SELECT CONCAT(first_name, ' ', last_name) AS full_name;
```

### LIMIT 和 OFFSET
```sql
-- MySQL
SELECT * FROM users LIMIT 10 OFFSET 20;

-- PostgreSQL（相同）
SELECT * FROM users LIMIT 10 OFFSET 20;
```

### 日期函数
```sql
-- MySQL
SELECT NOW(), DATE_FORMAT(created_at, '%Y-%m-%d');

-- PostgreSQL
SELECT NOW(), TO_CHAR(created_at, 'YYYY-MM-DD');
```

### IF 语句
```sql
-- MySQL
SELECT IF(age > 18, 'adult', 'minor') AS status;

-- PostgreSQL
SELECT CASE WHEN age > 18 THEN 'adult' ELSE 'minor' END AS status;
```

### 字符串转数字
```sql
-- MySQL
SELECT CAST('123' AS UNSIGNED);

-- PostgreSQL
SELECT CAST('123' AS INTEGER);
-- 或
SELECT '123'::INTEGER;
```

## 3. 索引差异

### 创建索引
```sql
-- MySQL
CREATE INDEX idx_name ON users(name);

-- PostgreSQL（相同）
CREATE INDEX idx_name ON users(name);
```

### 唯一索引
```sql
-- MySQL
CREATE UNIQUE INDEX idx_email ON users(email);

-- PostgreSQL（相同）
CREATE UNIQUE INDEX idx_email ON users(email);
```

### 全文搜索
- **MySQL**: `FULLTEXT` 索引
- **PostgreSQL**: `GIN` 或 `GiST` 索引（更强大）

## 4. 事务和锁

### 事务隔离级别
- **MySQL**: `READ UNCOMMITTED`, `READ COMMITTED`, `REPEATABLE READ`, `SERIALIZABLE`
- **PostgreSQL**: 相同，但默认是 `READ COMMITTED`

### 表锁
- **MySQL**: `LOCK TABLES`
- **PostgreSQL**: 使用行级锁，不需要表锁

## 5. 存储过程和函数

### 语法差异
```sql
-- MySQL
DELIMITER //
CREATE PROCEDURE get_user(IN user_id INT)
BEGIN
    SELECT * FROM users WHERE id = user_id;
END //
DELIMITER ;

-- PostgreSQL
CREATE OR REPLACE FUNCTION get_user(user_id INTEGER)
RETURNS TABLE(id INTEGER, name VARCHAR) AS $$
BEGIN
    RETURN QUERY SELECT * FROM users WHERE id = user_id;
END;
$$ LANGUAGE plpgsql;
```

## 6. 大小写敏感性

- **MySQL**: 表名和列名在 Linux 上区分大小写，Windows 上不区分
- **PostgreSQL**: 不区分大小写，但会转换为小写（除非用双引号）

```sql
-- PostgreSQL
CREATE TABLE Users (...);  -- 实际创建的是 users
CREATE TABLE "Users" (...);  -- 创建的是 Users（区分大小写）
```

## 7. 常用命令对比

| 操作 | MySQL | PostgreSQL |
|------|-------|------------|
| 显示数据库 | `SHOW DATABASES;` | `\l` 或 `\list` |
| 使用数据库 | `USE database;` | `\c database` |
| 显示表 | `SHOW TABLES;` | `\dt` |
| 显示表结构 | `DESC table;` | `\d table` |
| 退出 | `exit` 或 `quit` | `\q` |
| 显示帮助 | `help` | `\?` |

## 8. GORM 使用注意事项

### 1. 时间类型
```go
// 推荐使用 time.Time，GORM 会自动映射为 TIMESTAMPTZ
type User struct {
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### 2. JSON 类型
```go
// 使用 JSONB 类型
type User struct {
    Metadata datatypes.JSON `gorm:"type:jsonb"`
}
```

### 3. 自增主键
```go
// GORM 会自动使用 SERIAL
type User struct {
    ID uint `gorm:"primaryKey"`
}
```

### 4. 布尔类型
```go
// 直接使用 bool
type User struct {
    IsActive bool
}
```

## 9. 性能优化建议

1. **使用连接池**：PostgreSQL 的连接开销较大，使用连接池很重要
2. **索引策略**：PostgreSQL 的索引策略与 MySQL 类似，但支持更多索引类型
3. **查询计划**：使用 `EXPLAIN ANALYZE` 分析查询性能
4. **VACUUM**：定期运行 `VACUUM` 清理死元组（PostgreSQL 特有）

## 10. 常见陷阱

1. **字符串引号**：PostgreSQL 使用单引号表示字符串，双引号表示标识符
2. **NULL 比较**：使用 `IS NULL` 而不是 `= NULL`
3. **日期格式**：PostgreSQL 对日期格式要求更严格
4. **事务提交**：PostgreSQL 默认自动提交，需要显式使用事务

## 总结

从 MySQL 迁移到 PostgreSQL 需要注意：
- ✅ 数据类型映射（特别是自增主键）
- ✅ SQL 语法差异（字符串拼接、日期函数等）
- ✅ 大小写处理
- ✅ 事务和锁机制
- ✅ 序列的使用
- ✅ GORM 配置（时区、连接池等）

