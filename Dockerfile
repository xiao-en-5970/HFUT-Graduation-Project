# 运行预编译二进制的生产环境镜像（CI 中编译，不在此编译）
FROM ubuntu:24.04

# 安装时区数据
RUN apt-get update && apt-get install -y tzdata ca-certificates && rm -rf /var/lib/apt/lists/*

ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

WORKDIR /app

# 复制 CI 中已编译的 Linux 二进制（go build -o build/app）
COPY build/app ./

# 复制管理平台前端静态文件（/admin 路由）
COPY package/web/admin ./package/web/admin

# 环境变量由运行时 --env-file /opt/app/.env 或宿主环境传入
EXPOSE 8081

CMD ["./app"]
