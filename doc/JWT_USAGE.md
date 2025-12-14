# JWT 和中间件使用指南

## 功能概述

项目已实现：
1. **JWT 认证中间件** - 用于接口鉴权
2. **Zap 日志中间件** - 用于接口日志记录
3. **登录/注册返回 Token** - 用户认证后获得访问令牌

## JWT 配置

在 `.env` 文件中配置：

```env
JWT_SECRET=your-secret-key-change-in-production
JWT_EXPIRE_HOUR=24
```

**重要**：生产环境请务必修改 `JWT_SECRET` 为一个强随机字符串！

## API 使用方式

### 1. 用户注册

**请求：**
```bash
POST /api/v1/users/register
Content-Type: application/json

{
  "username": "testuser",
  "password": "password123",
  "school_id": 1
}
```

**响应：**
```json
{
  "code": 200,
  "message": "注册成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 1,
      "username": "testuser",
      ...
    }
  }
}
```

### 2. 用户登录

**请求：**
```bash
POST /api/v1/users/login
Content-Type: application/json

{
  "username": "testuser",
  "password": "password123"
}
```

**响应：**
```json
{
  "code": 200,
  "message": "登录成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 1,
      "username": "testuser",
      ...
    }
  }
}
```

### 3. 访问需要认证的接口

**请求头：**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**示例：**
```bash
GET /api/v1/articles
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

## 路由保护说明

### 不需要认证的路由（公开接口）
- `POST /api/v1/users/register` - 用户注册
- `POST /api/v1/users/login` - 用户登录
- `GET /health` - 健康检查

### 需要认证的路由（需要 JWT Token）
- 所有 `/api/v1/users` 的其他接口（获取、更新、列表）
- 所有 `/api/v1/articles` 接口
- 所有 `/api/v1/comments` 接口
- 所有 `/api/v1/likes` 接口
- 所有 `/api/v1/goods` 接口
- 所有 `/api/v1/tags` 接口
- 所有 `/api/v1/schools` 接口

## 中间件功能

### JWT 认证中间件 (`middleware.JWTAuth()`)

**功能：**
- 从请求头 `Authorization` 中提取 Bearer Token
- 验证 Token 的有效性
- 解析 Token 获取用户信息
- 将用户 ID 和用户名存储到 Gin 上下文

**使用方式：**
```go
// 在 controller 中获取用户ID
userID := middleware.GetUserID(ctx)
username := middleware.GetUsername(ctx)
```

### Zap 日志中间件 (`middleware.ZapLogger()`)

**功能：**
- 记录所有 API 请求的详细信息
- 包括：状态码、请求方法、路径、IP、耗时、用户代理等
- 根据状态码选择日志级别（Info/Warn/Error）
- 如果已认证，会记录用户 ID

**日志示例：**
```
INFO    HTTP Request    {"status": 200, "method": "GET", "path": "/api/v1/articles", "ip": "127.0.0.1", "latency": "12.5ms", "user_agent": "Mozilla/5.0...", "user_id": 1}
```

## 代码示例

### Controller 中获取当前用户

```go
func (c *ArticleController) Create(ctx *gin.Context) {
    // 从中间件获取用户ID
    userID := middleware.GetUserID(ctx)
    
    // 使用 userID 创建文章
    article, err := c.articleService.Create(userID, &req)
    // ...
}
```

### 权限验证示例

```go
func (c *UserController) Update(ctx *gin.Context) {
    id, _ := strconv.ParseUint(ctx.Param("id"), 10, 32)
    currentUserID := middleware.GetUserID(ctx)
    
    // 验证权限：只能修改自己的信息
    if currentUserID != uint(id) {
        ctx.JSON(http.StatusForbidden, response.Response{
            Code:    403,
            Message: "无权限访问此资源",
        })
        return
    }
    // ...
}
```

## 错误处理

### Token 相关错误

1. **未提供 Token**
   ```json
   {
     "code": 401,
     "message": "未提供认证 token"
   }
   ```

2. **Token 格式错误**
   ```json
   {
     "code": 401,
     "message": "认证 token 格式错误，应为: Bearer <token>"
   }
   ```

3. **Token 无效或过期**
   ```json
   {
     "code": 401,
     "message": "无效的 token: token is expired"
   }
   ```

## 安全建议

1. **生产环境配置**
   - 使用强随机字符串作为 `JWT_SECRET`
   - 建议使用至少 32 字符的随机字符串
   - 可以通过以下命令生成：
     ```bash
     openssl rand -base64 32
     ```

2. **Token 过期时间**
   - 根据业务需求设置合理的过期时间
   - 建议：普通用户 24 小时，管理员可适当延长

3. **HTTPS**
   - 生产环境务必使用 HTTPS
   - 防止 Token 在传输过程中被截获

4. **Token 刷新**
   - 可以考虑实现 Token 刷新机制
   - 在 Token 即将过期时自动刷新

## 测试示例

### 使用 curl 测试

```bash
# 1. 注册用户
curl -X POST http://localhost:8080/api/v1/users/register \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"123456"}'

# 2. 登录获取 Token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"123456"}' | jq -r '.data.token')

# 3. 使用 Token 访问受保护接口
curl -X GET http://localhost:8080/api/v1/articles \
  -H "Authorization: Bearer $TOKEN"
```

