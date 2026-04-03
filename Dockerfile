# 运行 CI/rsync 提供的预编译 Linux 二进制（build/app），不在镜像内编译 Go
FROM ubuntu:24.04

RUN apt-get update && apt-get install -y tzdata ca-certificates && rm -rf /var/lib/apt/lists/*

ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

WORKDIR /app
COPY build/app ./
COPY package/web/admin ./package/web/admin

EXPOSE 8081
CMD ["./app"]
