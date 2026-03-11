package client

import (
	"context"
	"fmt"
	"time"
)

// ChatClient 对话客户端
type ChatClient struct {
	client *Client
}

// ChatRequest 对话请求
type ChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
}

// ChatResponse 对话响应
type ChatResponse struct {
	Reply string `json:"reply"`
}

// Session 会话
type Session struct {
	ID        string
	CreatedAt time.Time
}

// Chat 智能对话(支持多轮)
// 通过sessionID管理对话上下文
func (c *ChatClient) Chat(ctx context.Context, message string, sessionID string) (*ChatResponse, error) {
	if message == "" {
		return nil, fmt.Errorf("message is required")
	}
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	req := &ChatRequest{
		Message:   message,
		SessionID: sessionID,
	}

	resp, err := c.client.post("/api/v1/chat/message", req)
	if err != nil {
		return nil, err
	}

	reply, ok := resp["reply"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return &ChatResponse{Reply: reply}, nil
}

// NewSession 创建新会话
// 返回一个唯一的会话ID
func (c *ChatClient) NewSession() *Session {
	return &Session{
		ID:        fmt.Sprintf("session-%d", time.Now().UnixNano()),
		CreatedAt: time.Now(),
	}
}

// ChatWithNewSession 创建新会话并发送消息
// 适合单轮对话场景
func (c *ChatClient) ChatWithNewSession(ctx context.Context, message string) (*ChatResponse, *Session, error) {
	session := c.NewSession()
	resp, err := c.Chat(ctx, message, session.ID)
	if err != nil {
		return nil, nil, err
	}
	return resp, session, nil
}

// ClearSession 清除会话
// 清除指定会话的历史记录
func (c *ChatClient) ClearSession(sessionID string) error {
	return c.client.delete(fmt.Sprintf("/api/v1/chat/session/%s", sessionID))
}

// MultiTurnChat 多轮对话
// 简化多轮对话的使用
func (c *ChatClient) MultiTurnChat(ctx context.Context, sessionID string, messages []string) ([]*ChatResponse, error) {
	responses := make([]*ChatResponse, len(messages))

	for i, msg := range messages {
		resp, err := c.Chat(ctx, msg, sessionID)
		if err != nil {
			return nil, fmt.Errorf("message %d failed: %w", i, err)
		}
		responses[i] = resp
	}

	return responses, nil
}

// ChatStream 对话流式响应（简化版）
// 通过回调函数处理响应
func (c *ChatClient) ChatStream(ctx context.Context, message string, sessionID string, onChunk func(chunk string)) error {
	// 注意：这里暂时使用普通Chat，因为服务端可能不支持流式响应
	// 如果需要真正的流式响应，需要服务端支持SSE
	resp, err := c.Chat(ctx, message, sessionID)
	if err != nil {
		return err
	}

	onChunk(resp.Reply)
	return nil
}

// ChatWithRetry 带重试的对话
// 在失败时自动重试
func (c *ChatClient) ChatWithRetry(ctx context.Context, message string, sessionID string, maxRetries int) (*ChatResponse, error) {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		resp, err := c.Chat(ctx, message, sessionID)
		if err == nil {
			return resp, nil
		}
		lastErr = err

		// 等待一段时间后重试
		time.Sleep(time.Duration(i+1) * time.Second)
	}

	return nil, fmt.Errorf("after %d retries, last error: %w", maxRetries, lastErr)
}

// Ask 便捷方法：提问
// 使用默认会话ID进行单次问答
func (c *ChatClient) Ask(ctx context.Context, question string) (*ChatResponse, error) {
	session := c.NewSession()
	return c.Chat(ctx, question, session.ID)
}

// AskWithContext 带上下文的提问
// 在已有会话中继续对话
func (c *ChatClient) AskWithContext(ctx context.Context, question string, sessionID string) (*ChatResponse, error) {
	return c.Chat(ctx, question, sessionID)
}
