.PHONY: run build test clean deps

# Go 参数
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod

# 构建参数
BINARY_NAME=hunyuan-api
BUILD_DIR=./build
MAIN_PACKAGE=./cmd/server

# 运行项目（自动加载 .env）
run: deps
	set -a && . ./.env && set +a && $(GOCMD) run $(MAIN_PACKAGE)/main.go

# 下载依赖
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# 构建二进制
build: deps
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

# 运行测试
test:
	$(GOTEST) -v ./...

# 清理构建文件
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
