# syntax=docker/dockerfile:1
# 多阶段：在镜像内编译，便于部署机单点 docker build 并复用本机 layer / BuildKit 缓存
FROM golang:1.23-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/app .

FROM ubuntu:24.04
RUN apt-get update && apt-get install -y tzdata ca-certificates && rm -rf /var/lib/apt/lists/*

ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

WORKDIR /app
COPY --from=builder /out/app ./
COPY package/web/admin ./package/web/admin

EXPOSE 8081
CMD ["./app"]
