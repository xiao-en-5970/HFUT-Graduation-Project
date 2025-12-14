# Makefile for HFUT Graduation Project

# 项目信息
PROJECT_NAME := HFUT-Graduation-Project
BINARY_NAME := app.exe
MAIN_PATH := ./main.go
BUILD_DIR := ./bin

# Go 参数
GO := go
GOBUILD := $(GO) build
GORUN := $(GO) run
GOTEST := $(GO) test
GOMOD := $(GO) mod

# 默认目标
.DEFAULT_GOAL := help

.PHONY: help
help: ## 显示帮助信息
	@echo "可用命令:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""

.PHONY: run
run: ## 运行项目（开发模式）
	@echo "正在启动项目..."
	$(GORUN) $(MAIN_PATH)

.PHONY: build
build: clean ## 构建项目二进制文件
	@echo "正在构建项目..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "构建完成: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: build-linux
build-linux: clean ## 构建 Linux 版本
	@echo "正在构建 Linux 版本..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "构建完成: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: build-darwin
build-darwin: clean ## 构建 macOS 版本
	@echo "正在构建 macOS 版本..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin $(MAIN_PATH)
	@echo "构建完成: $(BUILD_DIR)/$(BINARY_NAME)-darwin"

.PHONY: build-windows
build-windows: clean ## 构建 Windows 版本
	@echo "正在构建 Windows 版本..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-windows.exe $(MAIN_PATH)
	@echo "构建完成: $(BUILD_DIR)/$(BINARY_NAME)-windows.exe"

.PHONY: build-all
build-all: build-linux build-darwin build-windows ## 构建所有平台版本

.PHONY: clean
clean: ## 清理构建文件
	@echo "正在清理构建文件..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@rm -f $(BINARY_NAME).exe
	@echo "清理完成"

.PHONY: test
test: ## 运行测试
	@echo "正在运行测试..."
	$(GOTEST) -v ./...

.PHONY: test-coverage
test-coverage: ## 运行测试并生成覆盖率报告
	@echo "正在运行测试并生成覆盖率报告..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

.PHONY: deps
deps: ## 下载依赖
	@echo "正在下载依赖..."
	$(GOMOD) download
	$(GOMOD) tidy

.PHONY: deps-update
deps-update: ## 更新依赖
	@echo "正在更新依赖..."
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

.PHONY: fmt
fmt: ## 格式化代码
	@echo "正在格式化代码..."
	$(GO) fmt ./...

.PHONY: vet
vet: ## 运行 go vet 检查
	@echo "正在运行 go vet..."
	$(GO) vet ./...

.PHONY: lint
lint: vet ## 运行代码检查（vet）
	@echo "代码检查完成"

.PHONY: install
install: build ## 构建并安装到系统
	@echo "正在安装..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME) || echo "需要 sudo 权限才能安装到 /usr/local/bin"

.PHONY: dev
dev: run ## 开发模式运行（run 的别名）

.PHONY: check
check: fmt vet test ## 运行所有检查（格式化、vet、测试）

# Docker 相关变量
DOCKER_IMAGE := hfut-project
DOCKERFILE := package/docker/apiserver/Dockerfile
GOMOD_CACHE_DIR := $(shell pwd)/.docker-gomod-cache

.PHONY: docker-build
docker-build: ## 构建 Docker 镜像（使用 BuildKit 缓存）
	@echo "正在构建 Docker 镜像（使用 BuildKit 缓存）..."
	@echo "提示: 确保启用了 Docker BuildKit (DOCKER_BUILDKIT=1)"
	DOCKER_BUILDKIT=1 docker build \
		-f $(DOCKERFILE) \
		-t $(DOCKER_IMAGE) .
	@echo "构建完成: $(DOCKER_IMAGE)"

.PHONY: docker-run
docker-run: ## 运行 Docker 容器（映射端口 8081:8080）
	@echo "正在运行 Docker 容器..."
	docker run -d \
		-p 8081:8080 \
		--name $(DOCKER_IMAGE) \
		--restart unless-stopped \
		$(DOCKER_IMAGE)
	@echo "容器已启动，访问 http://localhost:8081"

.PHONY: docker-run-foreground
docker-run-foreground: ## 前台运行 Docker 容器（映射端口 8081:8080）
	@echo "正在前台运行 Docker 容器..."
	docker run --rm \
		-p 8081:8080 \
		$(DOCKER_IMAGE)

.PHONY: docker-stop
docker-stop: ## 停止 Docker 容器
	@echo "正在停止 Docker 容器..."
	-docker stop $(DOCKER_IMAGE) 2>/dev/null || true
	@echo "容器已停止"

.PHONY: docker-logs
docker-logs: ## 查看 Docker 容器日志
	docker logs -f $(DOCKER_IMAGE)

.PHONY: docker-build-no-cache
docker-build-no-cache: ## 构建 Docker 镜像（不使用缓存）
	@echo "正在构建 Docker 镜像（不使用缓存）..."
	docker build --no-cache -f $(DOCKERFILE) -t $(DOCKER_IMAGE) .
	@echo "构建完成: $(DOCKER_IMAGE)"

.PHONY: docker-clean
docker-clean: ## 清理 Docker 缓存目录
	@echo "正在清理 Docker Go module 缓存..."
	@rm -rf $(GOMOD_CACHE_DIR)
	@echo "清理完成"
