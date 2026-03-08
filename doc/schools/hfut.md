# 合肥工业大学（HFUT）对接文档

## 学校代码

`hfut`

## 登录流程

1. **CAS 登录**：`https://cas.hfut.edu.cn`
    - 获取 session → 验证码 → 加密盐 → checkUserIdenty → POST 登录 → 获取 ticket
    - 密码 AES-ECB 加密，盐从 `checkInitVercode` 的 `LOGIN_FLAVORING` cookie 获取

2. **EAM 登录**：获取教务系统 session
    - `GET cas.hfut.edu.cn/cas/login?service=http://jxglstu.hfut.edu.cn/eams5-student/neusoft-sso/login`
    - 跟随 302 至 jxglstu.hfut.edu.cn，获取 EAM session cookie

3. **学生信息**：`http://jxglstu.hfut.edu.cn/eams5-student/for-std/student-info`
    - 需 EAM session
    - 首次访问重定向至 `/info/{studentCode}`，解析 HTML 获取学生信息
    - 返回字段：studentId、usernameZh、usernameEn、sex、department、major、class、campus 等

## 绑定学校

用户绑定学校时需提供学校端账号密码，通过 `schools.Login` 验证后写入 `user_cert` 表：

- 接口：`POST /api/v1/user/bind/school`
- 请求体：`{ "school_id": 1, "username": "学号", "password": "密码" }`
- 流程：查学校 code → 调用学校登录 → 成功则 Upsert user_cert、更新 user.school_id

## 参考实现

- 毕业设计后端：`package/schools/hfut/`
- hfut-api：`src/modules/eam_studentInfo.ts`、`src/modules/login.ts`
