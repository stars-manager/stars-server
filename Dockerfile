# ============================================
# 多阶段构建 - 第一阶段：构建
# ============================================
# 使用明确的版本标签，避免 latest 的不可预测性
# golang:1.22-alpine3.19 是当前稳定版本
FROM golang:1.22-alpine3.19 AS builder

# 安装构建依赖
# git: 用于获取版本信息
# curl: 用于下载 Syft
RUN apk add --no-cache git curl

# 设置工作目录
WORKDIR /app

# 安装 SBOM 生成工具（syft）- 可选步骤，失败不影响构建
RUN curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin v1.0.0 || \
    echo "⚠️  Syft installation failed, skipping SBOM generation"

# ============================================
# 依赖层（利用 Docker 缓存）
# ============================================
# 优先复制依赖文件，如果依赖未变化则使用缓存
COPY go.mod go.sum ./
RUN go mod download

# ============================================
# 构建层
# ============================================
# 复制源代码
COPY . .

# 构建参数（版本信息）
# 这些参数在构建时通过 --build-arg 传入
ARG VERSION=dev         # 版本号，默认 dev
ARG GIT_COMMIT=unknown  # Git commit 哈希
ARG BUILD_TIME=unknown  # 构建时间

# 构建二进制文件
# CGO_ENABLED=0: 禁用 CGO，生成静态链接的可执行文件
# -ldflags: 在编译时注入版本信息到 Go 全局变量
RUN CGO_ENABLED=0 go build \
    -ldflags "-X server/pkg/version.Version=${VERSION} \
              -X server/pkg/version.GitCommit=${GIT_COMMIT} \
              -X server/pkg/version.BuildTime=${BUILD_TIME}" \
    -o server ./cmd/server

# 生成 SBOM（软件物料清单）- 可选步骤
# 提升供应链安全，便于漏洞追踪
RUN syft /app -o spdx-json=/app/sbom.spdx.json || \
    echo '{"sbom": "generation failed"}' > /app/sbom.spdx.json

# ============================================
# 多阶段构建 - 第二阶段：运行
# ============================================
# 使用明确的版本标签
FROM alpine:3.19

# 安装运行时依赖
# ca-certificates: HTTPS 连接所需的 CA 证书
# wget: 用于健康检查
RUN apk --no-cache add ca-certificates wget

# 创建非 root 用户和组（安全最佳实践）
RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件和 SBOM
# 只复制必要的文件，减小镜像体积
COPY --from=builder --chown=appuser:appgroup /app/server /app/server
COPY --from=builder --chown=appuser:appgroup /app/sbom.spdx.json /app/sbom.spdx.json

# 创建临时目录并设置权限（用于只读文件系统）
RUN mkdir -p /tmp/app && \
    chown -R appuser:appgroup /tmp/app

# 切换到非 root 用户
USER appuser

# 暴露端口
# 声明容器监听的端口（仅文档作用）
EXPOSE 8080

# ============================================
# 健康检查配置
# ============================================
# Docker 会定期执行健康检查命令
# 如果命令返回非 0 退出码，则标记容器为不健康
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 启动服务
CMD ["/app/server"]
