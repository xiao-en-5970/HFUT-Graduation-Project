
FROM ubuntu:latest

LABEL authors="dp"

# 安装必要的基础库（ca-certificates 用于 HTTPS 请求）
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# 从本地复制二进制文件
COPY ./bin/app.exe /app/app.exe

# 设置工作目录
WORKDIR /app

# 确保二进制文件有执行权限
RUN chmod +x /app/app.exe

# 运行应用
CMD ["./app.exe"]