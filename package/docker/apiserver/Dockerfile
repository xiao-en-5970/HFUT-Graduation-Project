FROM golang:1.23.7-alpine

LABEL authors="dp"

# 设置 Go module 缓存目录（用于挂载到本地）
ENV GOMODCACHE=/go/pkg/mod
ENV GOPATH=/go

# 设置工作目录
WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖（使用 BuildKit cache mount 加速构建）
# 如果需要在运行时挂载，使用 -v 参数挂载 /go/pkg/mod 到本地目录
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# 复制所有源代码
COPY . .

# 运行应用
CMD ["go", "run", "main.go"]