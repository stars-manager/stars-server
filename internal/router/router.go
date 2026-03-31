package router

import (
	"net/http"
	"runtime"

	"server/internal/handler"
	"server/pkg/constants"
	"server/pkg/middleware"
	"server/pkg/response"
	"server/pkg/version"
)

// Router 路由器
type Router struct {
	textHandler          *handler.TextHandler
	chatHandler          *handler.ChatHandler
	authHandler          *handler.AuthHandler
	githubHandler        *handler.GitHubHandler
	authMiddleware       func(http.Handler) http.Handler
	optionalAuthMiddleware func(http.Handler) http.Handler
}

// NewRouter 创建路由器
func NewRouter(
	textHandler *handler.TextHandler,
	chatHandler *handler.ChatHandler,
	authHandler *handler.AuthHandler,
	githubHandler *handler.GitHubHandler,
	authMiddleware func(http.Handler) http.Handler,
	optionalAuthMiddleware func(http.Handler) http.Handler,
) *Router {
	return &Router{
		textHandler:          textHandler,
		chatHandler:          chatHandler,
		authHandler:          authHandler,
		githubHandler:        githubHandler,
		authMiddleware:       authMiddleware,
		optionalAuthMiddleware: optionalAuthMiddleware,
	}
}

// Setup 设置路由
func (r *Router) Setup() *http.ServeMux {
	mux := http.NewServeMux()

	// ========== 认证相关路由 ==========
	mux.HandleFunc("GET /api/v1/auth/login", r.authHandler.Login)
	mux.HandleFunc("GET /api/v1/auth/callback", r.authHandler.Callback)
	mux.Handle("GET /api/v1/auth/user", middleware.Chain(http.HandlerFunc(r.authHandler.GetCurrentUser), r.optionalAuthMiddleware))
	mux.HandleFunc("POST /api/v1/auth/logout", r.authHandler.Logout)

	// ========== GitHub API 代理路由（需要认证）==========
	mux.Handle("GET /api/v1/github/user/repos", middleware.Chain(http.HandlerFunc(r.githubHandler.GetUserRepos), r.authMiddleware))
	mux.Handle("POST /api/v1/github/user/repos", middleware.Chain(http.HandlerFunc(r.githubHandler.CreateRepo), r.authMiddleware))
	mux.Handle("GET /api/v1/github/user/starred", middleware.Chain(http.HandlerFunc(r.githubHandler.GetUserStarred), r.authMiddleware))
	mux.Handle("GET /api/v1/github/repos/{owner}/{repo}/contents/{path}", middleware.Chain(http.HandlerFunc(r.githubHandler.GetFile), r.authMiddleware))
	mux.Handle("GET /api/v1/github/repos/{owner}/{repo}/contents/{path}/decoded", middleware.Chain(http.HandlerFunc(r.githubHandler.GetFileContent), r.authMiddleware))
	mux.Handle("PUT /api/v1/github/repos/{owner}/{repo}/contents/{path}", middleware.Chain(http.HandlerFunc(r.githubHandler.UpdateFile), r.authMiddleware))

	// ========== 原有路由 ==========
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
