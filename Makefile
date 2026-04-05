# Makefile - Go 项目构建自动化
# 使用 make 命令执行对应的目标

.PHONY: run build build-prod test clean deps

# ============================================
# Go 命令配置
# ============================================
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod

# ============================================
# 构建参数配置
# ============================================
BINARY_NAME=stars-server   # 二进制文件名称
BUILD_DIR=./build          # 构建输出目录
MAIN_PACKAGE=./cmd/server  # 主包路径

# 版本信息（从 Git 自动获取）
# VERSION: 从 Git 标签获取版本号，如 v1.0.0
# GIT_COMMIT: 当前 commit 的短哈希，如 abc123
# BUILD_TIME: 构建时间（UTC），如 2026-04-06T02:48:00Z
VERSION?=$(shell git describe --tags --always --dirty)
GIT_COMMIT=$(shell git rev-parse --short HEAD)
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# ============================================
# 开发命令
# ============================================

# 运行项目（开发模式）
# 自动加载 .env 环境变量并启动服务
run: deps
	set -a && . ./.env && set +a && $(GOCMD) run $(MAIN_PACKAGE)/main.go

# 下载并整理依赖
deps:
	$(GOMOD) download  # 下载依赖到缓存
	$(GOMOD) tidy      # 整理 go.mod 和 go.sum

# ============================================
# 构建命令
# ============================================

# 构建二进制（开发版）
# 不包含版本信息，快速构建用于本地测试
build: deps
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

# 构建生产版本（带版本信息）
# 使用 -ldflags 在编译时注入版本信息到二进制文件
# 这些信息可通过 /version API 查询
build-prod: deps
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -ldflags "-X server/pkg/version.Version=$(VERSION) \
		-X server/pkg/version.GitCommit=$(GIT_COMMIT) \
		-X server/pkg/version.BuildTime=$(BUILD_TIME)" \
		-o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

# ============================================
# 测试和清理
# ============================================

# 运行所有单元测试（详细输出）
test:
	$(GOTEST) -v ./...

# 清理构建产物
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

# ============================================
# Docker 命令
# ============================================

# 构建 Docker 镜像（开发版）
docker-build:
	docker build -t $(BINARY_NAME):dev .

# 构建 Docker 镜像（生产版，带版本信息）
docker-build-prod:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(BINARY_NAME):$(VERSION) \
		-t $(BINARY_NAME):latest \
		.

# 运行 Docker 容器（使用 docker-compose）
docker-run:
	docker-compose up -d

# 停止 Docker 容器
docker-stop:
	docker-compose down

# 查看 Docker 容器日志
docker-logs:
	docker-compose logs -f

# 清理 Docker 资源
docker-clean:
	docker-compose down -v
	docker image prune -f

# ============================================
# 代码质量
# ============================================

# 格式化代码
fmt:
	$(GOCMD) fmt ./...

# 代码检查（需要安装 golangci-lint）
lint:
	golangci-lint run ./...

# 安全检查（需要安装 gosec）
security:
	gosec ./...

# ============================================
# 发布相关
# ============================================

# 创建 Git 标签
tag:
	git tag -a v$(VERSION) -m "Release v$(VERSION)"
	git push origin v$(VERSION)

# 构建发布版本（包含所有检查）
release: test fmt build-prod docker-build-prod
	@echo "✅ 发布版本构建完成: $(VERSION)"

# ============================================
# 帮助信息
# ============================================

help:
	@echo "可用命令："
	@echo "  make run            - 运行开发服务器"
	@echo "  make build          - 构建开发版本"
	@echo "  make build-prod     - 构建生产版本（带版本信息）"
	@echo "  make test           - 运行单元测试"
	@echo "  make fmt            - 格式化代码"
	@echo "  make lint           - 代码检查"
	@echo "  make docker-build   - 构建 Docker 镜像（开发版）"
	@echo "  make docker-run     - 启动 Docker 容器"
	@echo "  make docker-stop    - 停止 Docker 容器"
	@echo "  make docker-logs    - 查看容器日志"
	@echo "  make release        - 构建发布版本"
