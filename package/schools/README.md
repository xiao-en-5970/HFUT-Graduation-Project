# 学校登录封装

每个学校一个目录，封装与学校端的认证对接。验证码由前端获取并填写，不再使用 OCR。

**关系**：`schools.code`（如 hfut）对应 `package/schools/hfut/`，绑定与认证时按 code 调用对应实现。

## 目录结构

```
package/schools/
├── school.go         # 接口定义
├── registry.go       # 注册与统一入口
├── captcha_session.go # 验证码会话 Redis 存储
├── README.md
└── hfut/             # 合肥工业大学
    ├── init.go       # 注册
    ├── login.go      # CAS 登录（需 captcha 参数）
    └── student_info.go # 学生信息拉取
```

## 新增学校

1. 在 `package/schools/` 下创建目录，如 `xxx/`
2. 实现 `School` 接口（Code、Name、Login）
3. 需验证码的学校实现 `SchoolWithCaptcha`（增加 GetCaptcha）
4. 在 `init()` 中调用 `schools.Register(&XXX{})`
5. 在 `main.go` 中 `import _ ".../package/schools/xxx"`

## 验证码流程

1. 前端调用 `GET /api/v1/schools/:id/captcha` 获取 base64 图片和 token
2. 用户填写验证码
3. 提交 bind/school 或 school-login 时传入 `captcha`、`captcha_token`
