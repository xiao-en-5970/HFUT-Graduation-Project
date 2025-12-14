FROM golang:1.23.7-alpine


# 设置 Go module 缓存目录
ENV GOMODCACHE=/var/modtmp
ENV GOPATH=/go

# 设置 Go 中国代理
ENV GOPROXY=https://goproxy.cn,direct

# 设置工作目录
WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖（缓存会保存到 /var/modtmp，通过 volume 挂载）
RUN go mod download

# 复制所有源代码
COPY . .

# 运行应用
CMD ["go", "run", "."]