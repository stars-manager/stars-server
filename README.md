# Hunyuan API Server

基于腾讯混元大模型的 GitHub Stars 项目标签生成和智能对话 RESTful API

## 快速开始

```bash
cp .env.example .env
vim .env  # 填写 HUNYUAN_API_KEY
make run
```

Docker 部署：

```bash
cp .env.example .env && vim .env
docker-compose up -d
```

---

## REST API

### 1. 项目标签生成

为 GitHub Stars 项目生成总结和标签。

```
POST /api/v1/stars/tags
```

**请求：**

```json
{
  "projects": [
    {
      "name": "Vue.js",
      "full_name": "vuejs/vue",
      "description": "渐进式 JavaScript 框架",
      "language": "TypeScript",
      "url": "https://github.com/vuejs/vue",
      "stars": 207000,
      "forks": 33600,
      "topics": ["javascript", "framework", "frontend"]
    }
  ]
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `projects` | array | 是 | 项目列表（最多 20 个） |
| `projects[].name` | string | 是 | 项目名称 |
| `projects[].full_name` | string | 否 | 完整名称 owner/repo |
| `projects[].description` | string | 否 | 项目描述 |
| `projects[].language` | string | 否 | 主要编程语言 |
| `projects[].url` | string | 否 | 项目地址 |
| `projects[].stars` | int | 否 | Star 数 |
| `projects[].forks` | int | 否 | Fork 数 |
| `projects[].topics` | array | 否 | 主题标签 |

**响应：**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "projects": [
      {
        "name": "Vue.js",
        "summary": "渐进式 JavaScript 前端框架",
        "tags": ["前端框架", "JavaScript"]
      },
      {
        "name": "React",
        "summary": "Facebook 开源的组件化 UI 库",
        "tags": ["前端框架", "UI组件"]
      }
    ],
    "project_count": 2,
    "process_time": 1200000000
  }
}
```

---

### 2. 智能对话

支持多轮对话，可选附带文档进行问答。

```
POST /api/v1/chat/message
```

**普通对话：**

```json
{
  "message": "推荐一些前端框架",
  "session_id": "user-123"
}
```

**带文档问答：**

```json
{
  "message": "这两个框架有什么区别？",
  "session_id": "user-123",
  "documents": [
    "Vue.js 是渐进式框架，采用双向数据绑定...",
    "React 是组件化 UI 库，采用单向数据流..."
  ]
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `message` | string | 是 | 用户消息（1-5000 字符） |
| `session_id` | string | 是 | 会话 ID（4-64 字符，仅字母数字下划线连字符） |
| `documents` | array | 否 | 文档列表（最多 10 篇，每篇最多 5000 字符） |

**响应：**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "reply": "推荐以下前端框架：Vue.js、React、Angular...",
    "session_id": "user-123",
    "is_new_session": false,
    "has_documents": false,
    "message_count": 4,
    "process_time": 2500000000
  }
}
```

---

### 3. 清除会话

```
DELETE /api/v1/chat/session/{session_id}
```

**响应：**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "message": "session cleared successfully",
    "session_id": "user-123",
    "total_sessions": 5
  }
}
```

---

### 4. 健康检查

```
GET /health
```

**响应：**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "status": "ok",
    "service": "hunyuan-api",
    "runtime": {
      "goroutines": 15,
      "go_version": "go1.21.5",
      "platform": "darwin/arm64"
    }
  }
}
```

---

### 5. 版本信息

```
GET /version
```

**响应：**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "version": "1.0.0",
    "git_commit": "abc123",
    "build_time": "2024-01-01T00:00:00Z",
    "go_version": "go1.21.5",
    "platform": "darwin/arm64"
  }
}
```

---

## 错误码

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 1001 | 请求格式错误 |
| 1002 | 参数校验失败 |
| 1003 | JSON 解析失败 |
| 2001 | 服务器内部错误 |
| 2002 | LLM 调用失败 |
| 2006 | 会话历史过长，请开启新会话 |

---

## 配置

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `HUNYUAN_API_KEY` | - | **必填**，[获取地址](https://console.cloud.tencent.com/hunyuan) |
| `HUNYUAN_MODEL` | `hunyuan-lite` | 可选: lite/standard/pro/turbo |
| `HUNYUAN_BASE_URL` | `https://api.hunyuan.cloud.tencent.com/v1` | API 地址 |
| `PORT` | `8080` | 服务端口 |

## 构建

```bash
# 开发构建
make run

# 生产构建（带版本信息）
VERSION=$(git describe --tags --always)
GIT_COMMIT=$(git rev-parse --short HEAD)
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build -ldflags "-X server/pkg/version.Version=$VERSION \
  -X server/pkg/version.GitCommit=$GIT_COMMIT \
  -X server/pkg/version.BuildTime=$BUILD_TIME" \
  -o bin/server ./cmd/server
```

## 项目结构

```
.
├── cmd/server/          # 应用入口
├── internal/
│   ├── handler/         # HTTP 处理器
│   └── router/          # 路由配置
├── pkg/
│   ├── client/          # API 客户端 SDK
│   ├── config/          # 配置管理
│   ├── constants/       # 常量定义
│   ├── llm/             # LLM 客户端
│   ├── middleware/      # HTTP 中间件
│   ├── response/        # 响应处理
│   ├── service/         # 业务逻辑
│   ├── utils/           # 工具函数
│   └── version/         # 版本管理
├── go.mod
├── Makefile
└── README.md
```

## License

Apache 2.0
