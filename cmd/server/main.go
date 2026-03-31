package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"server/internal/handler"
	"server/internal/router"
	"server/pkg/auth"
	"server/pkg/config"
	githubclient "server/pkg/github"
	"server/pkg/llm"
	"server/pkg/middleware"
	"server/pkg/service/chat"
	"server/pkg/service/text"
)

func main() {
	// 加载 .env 文件（开发环境）
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	cfg := config.Load()

	// 验证配置
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Config validation failed: %v", err)
	}

	// ========== 初始化认证组件 ==========
	// OAuth 客户端
	oauthClient := auth.NewOAuthClient(
		cfg.GitHub.ClientID,
		cfg.GitHub.ClientSecret,
		cfg.GitHub.RedirectURI,
	)

	// Session 管理器
	sessionManager, err := auth.NewSessionManager(
		cfg.Session.Secret,
		cfg.Session.EncryptionKey,
		cfg.Session.MaxAge,
		cfg.Session.Secure,
	)
	if err != nil {
		log.Fatalf("Failed to create session manager: %v", err)
	}

	// 认证中间件
	authMiddleware := auth.Auth(sessionManager)
	optionalAuthMiddleware := auth.OptionalAuth(sessionManager)

	// GitHub API 客户端
	githubClient := githubclient.NewClient()

	// ========== 初始化原有服务 ==========
	llmClient := llm.NewClient(&cfg.LLM)
	textHandler := handler.NewTextHandler(text.NewService(llmClient))
	chatHandler := handler.NewChatHandler(chat.NewService(llmClient))

	// ========== 初始化新处理器 ==========
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	authHandler := handler.NewAuthHandler(oauthClient, sessionManager, frontendURL)
	githubHandler := handler.NewGitHubHandler(githubClient)

	// ========== 设置路由 ==========
	mux := router.NewRouter(
		textHandler,
		chatHandler,
		authHandler,
		githubHandler,
		authMiddleware,
		optionalAuthMiddleware,
	).Setup()

	// 应用中间件：恢复 -> 请求ID -> 日志 -> CORS
	handler := middleware.Chain(mux,
		middleware.Recover,
		middleware.RequestID,
		middleware.Logger,
		middleware.CORS,
	)

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Printf("Server listening on :%s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// 启动会话清理 goroutine（每 5 分钟清理一次过期会话）
	go chat.StartSessionCleanup(5 * time.Minute)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	} else {
		log.Println("Server stopped gracefully")
	}
}
