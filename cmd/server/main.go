package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"server/internal/handler"
	"server/internal/router"
	"server/pkg/config"
	"server/pkg/llm"
	"server/pkg/middleware"
	"server/pkg/service/chat"
	"server/pkg/service/text"
)

func main() {
	cfg := config.Load()

	// 验证配置
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Config validation failed: %v", err)
	}

	llmClient := llm.NewClient(&cfg.LLM)
	textHandler := handler.NewTextHandler(text.NewService(llmClient))
	chatHandler := handler.NewChatHandler(chat.NewService(llmClient))
	mux := router.NewRouter(textHandler, chatHandler).Setup()

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
