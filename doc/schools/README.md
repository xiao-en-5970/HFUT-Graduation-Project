# 学校对接接口文档

本目录记录各学校对接的接口说明，供开发与联调参考。

## 目录结构

每个学校有独立子目录，内含：

- `README.md`：对接流程、登录说明、绑定说明
- `openapi.json`：该学校相关的 API 定义（school-login、bind/school 等）

```
doc/schools/
├── README.md          # 本文件
├── hfut/              # 合肥工业大学
│   ├── README.md
│   └── openapi.json
└── {code}/             # 其他学校，code 如 school_code
    ├── README.md
    └── openapi.json
```

## 已对接学校

| 代码 | 学校 | 文档 |
|-----|------|------|
| hfut | 合肥工业大学 | [hfut/](./hfut/) |

## 通用流程

1. 用户绑定学校前，需通过学校端登录验证（账号+密码）
2. 验证成功后，拉取学生信息并存入 `user_cert` 表
3. 绑定成功后，用户获得该学校的访问权限
