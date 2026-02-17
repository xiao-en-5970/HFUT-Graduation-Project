# 运行预编译二进制的生产环境镜像（CI 中编译，不在此编译）
FROM ubuntu:24.04

# 安装时区数据
RUN apt-get update && apt-get install -y tzdata ca-certificates && rm -rf /var/lib/apt/lists/*

ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

WORKDIR /app

# 复制 CI 中已编译的 Linux 二进制（go build -o build/app）
COPY build/app ./

# 应用端口（与 bootstrap 中 config.ServerPort 对应）
ENV SERVER_PORT=8081
EXPOSE 8081

CMD ["./app"]
