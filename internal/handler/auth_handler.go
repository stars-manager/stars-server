package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"server/pkg/auth"
	"server/pkg/response"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	oauthClient     *auth.OAuthClient
	sessionManager  *auth.SessionManager
	frontendURL     string
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(oauthClient *auth.OAuthClient, sessionManager *auth.SessionManager, frontendURL string) *AuthHandler {
	return &AuthHandler{
		oauthClient:    oauthClient,
		sessionManager: sessionManager,
		frontendURL:    frontendURL,
	}
}

// LoginResponse 登录响应
type LoginResponse struct {
	AuthURL string `json:"auth_url"`
}

// Login 获取 OAuth 授权 URL
// GET /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// 生成随机 state（用于防止 CSRF）
	state := generateState()

	// 生成授权 URL
	authURL := h.oauthClient.GetAuthURL(state)

	// 可以将 state 存储到 session 或 cache 中验证（简化版暂不实现）

	response.Success(w, LoginResponse{
		AuthURL: authURL,
	})
}

// Callback OAuth 回调处理
// GET /api/v1/auth/callback
func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	// state := r.URL.Query().Get("state") // TODO: 验证 state 防止 CSRF

	if code == "" {
		response.Error(w, response.CodeBadRequest, "missing authorization code")
		return
	}

	// 验证 state（简化版暂不实现）

	// 完成 OAuth 认证
	user, accessToken, err := h.oauthClient.Authenticate(r.Context(), code)
	if err != nil {
		response.Error(w, response.CodeInternalError, fmt.Sprintf("authentication failed: %v", err))
		return
	}

	// 创建 session
	userID := fmt.Sprintf("%d", user.ID)
	tokenString, err := h.sessionManager.CreateSession(userID, user.Login, user.AvatarURL, accessToken)
	if err != nil {
		response.Error(w, response.CodeInternalError, "failed to create session")
		return
	}

	// 设置 Cookie
	h.sessionManager.SetCookie(w, tokenString)

	// 重定向回前端首页
	http.Redirect(w, r, h.frontendURL, http.StatusTemporaryRedirect)
}

// UserResponse 用户信息响应
type UserResponse struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
}

// GetCurrentUser 获取当前用户信息
// GET /api/v1/auth/user
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	// 从 context 中获取用户信息（由中间件注入）
	userCtx := auth.GetUserFromContext(r.Context())
	if userCtx == nil {
		// 未登录，返回 401
		response.Error(w, response.CodeUnauthorized, "not authenticated")
		return
	}

	response.Success(w, UserResponse{
		ID:        userCtx.ID,
		Username:  userCtx.Username,
		AvatarURL: userCtx.AvatarURL,
	})
}

// Logout 登出
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// 清除 Cookie
	h.sessionManager.ClearCookie(w)

	response.Success(w, map[string]string{
		"message": "logged out successfully",
	})
}

// generateState 生成随机 state
func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// writeJSON 写入 JSON 响应（辅助函数）
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
