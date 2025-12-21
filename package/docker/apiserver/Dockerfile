FROM ubuntu:latest

# 安装时区数据
RUN apt-get update && apt-get install -y tzdata ca-certificates && rm -rf /var/lib/apt/lists/*

# 设置时区为 Asia/Shanghai（解决 PostgreSQL 时区问题）
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 设置工作目录
WORKDIR /app

# 复制二进制文件
COPY bin/app.exe ./bin/app.exe

# 复制 .env 文件
COPY .env ./

# 运行应用
CMD ["./bin/app.exe"]
