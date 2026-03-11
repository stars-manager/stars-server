package handler

import (
	"encoding/json"
	"net/http"

	"server/pkg/response"
	"server/pkg/service/chat"
	"server/pkg/service/text"
)

// TextHandler 文本处理处理器
type TextHandler struct {
	textService *text.Service
}

// NewTextHandler 创建文本处理器
func NewTextHandler(textService *text.Service) *TextHandler {
	return &TextHandler{
		textService: textService,
	}
}

// StarsTags 处理项目标签请求
// POST /api/v1/stars/tags
func (h *TextHandler) StarsTags(w http.ResponseWriter, r *http.Request) {
	// 限制请求体大小（1MB）
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req text.StarsTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, response.CodeInvalidJSON, "请求格式错误: "+err.Error())
		return
	}

	resp, err := h.textService.StarsTags(r.Context(), &req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	response.Success(w, resp)
}

// ChatHandler 对话处理器
type ChatHandler struct {
	chatService *chat.Service
}

// NewChatHandler 创建对话处理器
func NewChatHandler(chatService *chat.Service) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
	}
}

// Chat 处理对话请求
// POST /api/v1/chat/message
func (h *ChatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	// 限制请求体大小（1MB）
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req chat.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, response.CodeInvalidJSON, "请求格式错误: "+err.Error())
		return
	}

	resp, err := h.chatService.Chat(r.Context(), &req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	response.Success(w, resp)
}

// ClearSession 清除会话
// DELETE /api/v1/chat/session/{session_id}
func (h *ChatHandler) ClearSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("session_id")
	if sessionID == "" {
		response.Error(w, response.CodeInvalidParam, "session_id is required")
		return
	}

	err := h.chatService.ClearSession(sessionID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	// 返回清除结果和统计信息
	stats := h.chatService.GetSessionStats()

	response.Success(w, map[string]any{
		"message":        "session cleared successfully",
		"session_id":     sessionID,
		"total_sessions": stats["total_sessions"],
	})
}

// handleServiceError 统一处理服务层错误
func handleServiceError(w http.ResponseWriter, err error) {
	if svcErr, ok := err.(*response.ServiceError); ok {
		response.Error(w, svcErr.Code, svcErr.Message)
		return
	}
	response.Error(w, response.CodeInternalError, err.Error())
}
