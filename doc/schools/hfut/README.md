# 合肥工业大学（HFUT）对接文档

## 学校代码

`hfut`

## 表单配置（必须配置，禁止写死）

- **form_fields**: `[{key:"username",label_zh:"学号",label_en:"Student ID"}, {key:"password",label_zh:"密码",label_en:"Password"}, {key:"captcha",label_zh:"验证码",label_en:"Captcha"}]`
- **login_url**: 必填，CAS 登录页地址，如 `https://cas.hfut.edu.cn/cas/login?service=...`
- **captcha_url**: 必填，验证码图片地址，如 `https://cas.hfut.edu.cn/cas/vercode`

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

1. **获取学校列表**：`GET /api/v1/schools` → 含 form_fields、captcha_url
2. **获取验证码**（form_fields 含 captcha 时）：`GET /api/v1/schools/:id/captcha` → `{ image, token }`
3. **提交绑定**：`POST /api/v1/user/bind/school`
   - 请求体：`{ "school_id": 1, "username": "学号", "password": "密码", "captcha": "验证码", "captcha_token": "xxx" }`
   - captcha、captcha_token 在 form_fields 含 captcha 时必填

## API 文档

详见 [openapi.json](./openapi.json)

## 参考实现

- 毕业设计后端：`package/schools/hfut/`
- hfut-api：`src/modules/eam_studentInfo.ts`、`src/modules/login.ts`
