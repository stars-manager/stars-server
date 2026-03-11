package chat

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/sashabaranov/go-openai"
	"server/pkg/llm"
	"server/pkg/response"
)

// 常量配置
const (
	MaxDocuments       = 10               // 最大文档数量
	MaxDocLength       = 5000             // 单个文档最大长度
	MaxMessageLength   = 5000             // 消息最大长度
	MinMessageLength   = 1                // 消息最小长度
	MaxHistoryMessages = 20               // 最大历史消息数
	MinSessionIDLength = 4                // SessionID最小长度
	MaxSessionIDLength = 64               // SessionID最大长度
	SessionTimeout     = 30 * time.Minute // 会话超时时间
)

// sessionID格式：字母、数字、连字符、下划线
var sessionIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Service 对话服务
type Service struct {
	client *llm.Client
}

// NewService 创建对话服务
func NewService(client *llm.Client) *Service {
	return &Service{
		client: client,
	}
}

// ChatRequest 对话请求
type ChatRequest struct {
	Message   string   `json:"message"`             // 用户消息（必填）
	SessionID string   `json:"session_id"`          // 会话ID（必填）
	Documents []string `json:"documents,omitempty"` // 文档列表（可选）
}

// Validate 验证请求
func (r *ChatRequest) Validate() error {
	// 验证消息
	r.Message = strings.TrimSpace(r.Message)
	if len(r.Message) < MinMessageLength {
		return fmt.Errorf("消息内容不能为空")
	}
	if len(r.Message) > MaxMessageLength {
		return fmt.Errorf("消息内容不能超过%d个字符", MaxMessageLength)
	}

	// 验证SessionID
	r.SessionID = strings.TrimSpace(r.SessionID)
	if len(r.SessionID) < MinSessionIDLength {
		return fmt.Errorf("会话ID长度不能少于%d个字符", MinSessionIDLength)
	}
	if len(r.SessionID) > MaxSessionIDLength {
		return fmt.Errorf("会话ID长度不能超过%d个字符", MaxSessionIDLength)
	}
	if !sessionIDRegex.MatchString(r.SessionID) {
		return fmt.Errorf("会话ID只能包含字母、数字、连字符和下划线")
	}

	// 验证文档（可选）
	if len(r.Documents) > MaxDocuments {
		return fmt.Errorf("文档数量不能超过%d篇", MaxDocuments)
	}
	for i, doc := range r.Documents {
		doc = strings.TrimSpace(doc)
		r.Documents[i] = doc
		if len(doc) > MaxDocLength {
			return fmt.Errorf("第%d篇文档长度超出限制", i+1)
		}
		if len(doc) == 0 {
			return fmt.Errorf("第%d篇文档内容为空", i+1)
		}
	}

	return nil
}

// ChatResponse 对话响应
type ChatResponse struct {
	Reply        string        `json:"reply"`          // AI回复
	SessionID    string        `json:"session_id"`     // 会话ID
	IsNewSession bool          `json:"is_new_session"` // 是否新会话
	HasDocuments bool          `json:"has_documents"`  // 是否有文档
	MessageCount int           `json:"message_count"`  // 当前会话消息数
	ProcessTime  time.Duration `json:"process_time"`   // 处理耗时
}

// Session 会话
type Session struct {
	Messages   []openai.ChatCompletionMessage
	LastActive time.Time
	mu         sync.RWMutex
}

// SessionManager 会话管理器
type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewSessionManager 创建会话管理器
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// GetSession 获取或创建会话
func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		now := time.Now()
		session = &Session{
			Messages:   []openai.ChatCompletionMessage{},
			LastActive: now,
		}
		sm.sessions[sessionID] = session
		return session, false // 返回 false 表示新创建
	}

	session.LastActive = time.Now()
	return session, true // 返回 true 表示已存在
}

// ClearSession 清除会话
func (sm *SessionManager) ClearSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.sessions[sessionID]; !exists {
		return fmt.Errorf("会话不存在")
	}
	delete(sm.sessions, sessionID)
	return nil
}

// Exists 检查会话是否存在
func (sm *SessionManager) Exists(sessionID string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	_, exists := sm.sessions[sessionID]
	return exists
}

// GetStats 获取会话统计
func (sm *SessionManager) GetStats() map[string]int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return map[string]int{
		"total_sessions": len(sm.sessions),
	}
}

// CleanupExpired 清理过期会话
func (sm *SessionManager) CleanupExpired() int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	count := 0
	for id, session := range sm.sessions {
		if time.Since(session.LastActive) > SessionTimeout {
			delete(sm.sessions, id)
			count++
		}
	}
	return count
}

// AddMessage 添加消息到会话
func (s *Session) AddMessage(role, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Messages = append(s.Messages, openai.ChatCompletionMessage{
		Role:    role,
		Content: content,
	})

	// 限制历史长度，保留最近的消息
	if len(s.Messages) > MaxHistoryMessages {
		s.Messages = s.Messages[len(s.Messages)-MaxHistoryMessages:]
	}
}

// GetMessages 获取消息历史
func (s *Session) GetMessages() []openai.ChatCompletionMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]openai.ChatCompletionMessage, len(s.Messages))
	copy(result, s.Messages)
	return result
}

// MessageCount 获取消息数量
func (s *Session) MessageCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.Messages)
}

var globalSessionManager = NewSessionManager()

// Chat 智能对话（支持多轮，可选文档）
func (s *Service) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	startTime := time.Now()

	// 验证输入
	if err := req.Validate(); err != nil {
		return nil, response.NewError(response.CodeInvalidParam, err.Error())
	}

	// 获取或创建会话
	session, isNewSession := globalSessionManager.GetSession(req.SessionID)

	// 检查会话历史长度
	if session.MessageCount() >= MaxHistoryMessages {
		return nil, response.NewError(response.CodeSessionTooLong)
	}

	// 构建用户消息
	userMessage := req.Message
	hasDocuments := len(req.Documents) > 0

	if hasDocuments {
		// 带文档：构建上下文
		var contextBuilder strings.Builder
		for i, doc := range req.Documents {
			fmt.Fprintf(&contextBuilder, "【文档%d】\n%s\n\n", i+1, doc)
		}
		userMessage = fmt.Sprintf("参考文档：\n%s\n问题：%s", contextBuilder.String(), req.Message)
	}

	// 添加用户消息到会话
	session.AddMessage(openai.ChatMessageRoleUser, userMessage)

	// 选择系统提示词
	var systemPrompt string
	if hasDocuments {
		systemPrompt = `你是一个专业的文档分析助手。
你的职责：
1. 优先基于用户提供的文档内容回答问题
2. 如果文档中没有相关信息，明确说明后再结合你的知识补充
3. 回答要准确、有条理、简洁
4. 引用文档内容时，标注文档编号（如：根据文档1...）
5. 如果是多轮对话，记住之前的讨论内容`
	} else {
		systemPrompt = `你是一个专业的技术助手，擅长项目推荐、技术讲解和问题讨论。
你的职责：
1. 项目推荐：根据用户需求推荐合适的技术项目或工具
2. 技术讲解：用通俗易懂的语言解释技术概念
3. 问题讨论：深入分析技术问题，提供专业见解
4. 保持友好、专业、诚实的态度
5. 如果是多轮对话，记住之前的讨论内容，保持对话连贯性
6. 对于不确定的内容，诚实说明你的局限性`
	}

	// 调用模型
	result, err := s.client.ChatCompletionWithSystem(ctx, systemPrompt, session.GetMessages(), llm.WithTemperature(0.7))
	if err != nil {
		return nil, response.NewError(response.CodeLLMError, fmt.Sprintf("对话失败: %v", err))
	}

	// 添加助手回复到会话历史
	session.AddMessage(openai.ChatMessageRoleAssistant, result)

	return &ChatResponse{
		Reply:        strings.TrimSpace(result),
		SessionID:    req.SessionID,
		IsNewSession: isNewSession,
		HasDocuments: hasDocuments,
		MessageCount: session.MessageCount(),
		ProcessTime:  time.Since(startTime),
	}, nil
}

// ClearSession 清除会话
func (s *Service) ClearSession(sessionID string) error {
	return globalSessionManager.ClearSession(sessionID)
}

// GetSessionStats 获取会话统计
func (s *Service) GetSessionStats() map[string]int {
	return globalSessionManager.GetStats()
}

// StartSessionCleanup 启动定期清理过期会话的任务
func StartSessionCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		count := globalSessionManager.CleanupExpired()
		if count > 0 {
			log.Printf("[Session Cleanup] Removed %d expired sessions", count)
		}
	}
}
