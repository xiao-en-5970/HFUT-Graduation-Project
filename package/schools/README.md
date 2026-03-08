# 学校登录封装

每个学校一个目录，封装与学校端的认证对接。用户只需输入**账号、密码**，验证码等由后端自动处理。

## 目录结构

```
package/schools/
├── school.go      # 接口定义
├── registry.go    # 注册与统一入口
├── README.md
└── hfut/          # 合肥工业大学
    ├── init.go    # 注册
    ├── login.go   # CAS 登录逻辑
    ├── captcha.go # 验证码 OCR（需 -tags tesseract）
    └── captcha_stub.go
```

## 新增学校

1. 在 `package/schools/` 下创建目录，如 `xxx/`
2. 实现 `School` 接口（Code、Name、Login）
3. 在 `init()` 中调用 `schools.Register(&XXX{})`
4. 在 `main.go` 中 `import _ ".../package/schools/xxx"`

## HFUT 验证码

默认编译不包含 OCR，需安装 tesseract 并加编译标签：

```bash
# macOS
brew install tesseract
go build -tags tesseract -o app .

# Linux
sudo apt install tesseract-ocr tesseract-ocr-eng
go build -tags tesseract -o app .
```

若不使用 `-tags tesseract`，学校登录会返回「需要 tesseract」错误。
