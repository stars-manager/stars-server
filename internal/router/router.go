package router

import (
	"net/http"
	"runtime"

	"server/internal/handler"
	"server/pkg/constants"
	"server/pkg/response"
	"server/pkg/version"
)

// Router 路由器
type Router struct {
	textHandler *handler.TextHandler
	chatHandler *handler.ChatHandler
}

// NewRouter 创建路由器
func NewRouter(textHandler *handler.TextHandler, chatHandler *handler.ChatHandler) *Router {
	return &Router{
		textHandler: textHandler,
		chatHandler: chatHandler,
	}
}

// Setup 设置路由
func (r *Router) Setup() *http.ServeMux {
	mux := http.NewServeMux()

	// 项目标签接口
	mux.HandleFunc("POST /api/v1/stars/tags", r.textHandler.StarsTags)

	// 对话接口
	mux.HandleFunc("POST /api/v1/chat/message", r.chatHandler.Chat)
	mux.HandleFunc("DELETE /api/v1/chat/session/{session_id}", r.chatHandler.ClearSession)

	// 健康检查
	mux.HandleFunc("GET /health", healthCheck)

	// 版本信息
	mux.HandleFunc("GET /version", versionInfo)

	return mux
}

// healthCheck 健康检查
func healthCheck(w http.ResponseWriter, r *http.Request) {
	response.Success(w, map[string]any{
		"status":  "ok",
		"service": constants.ServiceName,
		"runtime": map[string]any{
			"goroutines": runtime.NumGoroutine(),
			"go_version": runtime.Version(),
			"platform":   runtime.GOOS + "/" + runtime.GOARCH,
		},
	})
}

// versionInfo 版本信息
func versionInfo(w http.ResponseWriter, r *http.Request) {
	response.Success(w, version.Get())
}
