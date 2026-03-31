package auth

import (
	"context"
	"encoding/json"
	"net/http"

	"server/pkg/response"
)

// contextKey context 键类型
type contextKey string

const (
	// UserKey 用户信息 context key
	UserKey contextKey = "user"
	// TokenKey access token context key
	TokenKey contextKey = "token"
)

// UserContext 用户上下文信息
type UserContext struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
}

// Auth 认证中间件
func Auth(sessionManager *SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 从请求中获取 session
			claims, err := sessionManager.GetSessionFromRequest(r)
			if err != nil {
				response.Error(w, response.CodeUnauthorized, "unauthorized")
				return
			}

			// 解密 access token
			accessToken, err := sessionManager.GetAccessToken(claims)
			if err != nil {
				response.Error(w, response.CodeUnauthorized, "invalid session")
				return
			}

			// 创建用户上下文
			userCtx := &UserContext{
				ID:        0, // 将从 username 中解析，或添加到 claims
				Username:  claims.Username,
				AvatarURL: claims.AvatarURL,
			}

			// 将用户信息和 token 注入到 context
			ctx := r.Context()
			ctx = context.WithValue(ctx, UserKey, userCtx)
			ctx = context.WithValue(ctx, TokenKey, accessToken)

			// 调用下一个处理器
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserFromContext 从 context 中获取用户信息
func GetUserFromContext(ctx context.Context) *UserContext {
	user, _ := ctx.Value(UserKey).(*UserContext)
	return user
}

// GetTokenFromContext 从 context 中获取 access token
func GetTokenFromContext(ctx context.Context) string {
	token, _ := ctx.Value(TokenKey).(string)
	return token
}

// OptionalAuth 可选认证中间件（不强制要求登录）
func OptionalAuth(sessionManager *SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 尝试从请求中获取 session
			claims, err := sessionManager.GetSessionFromRequest(r)
			if err != nil {
				// 没有登录，继续执行但不注入用户信息
				next.ServeHTTP(w, r)
				return
			}

			// 解密 access token
			accessToken, err := sessionManager.GetAccessToken(claims)
			if err != nil {
				// token 无效，继续执行但不注入用户信息
				next.ServeHTTP(w, r)
				return
			}

			// 创建用户上下文
			userCtx := &UserContext{
				Username:  claims.Username,
				AvatarURL: claims.AvatarURL,
			}

			// 将用户信息和 token 注入到 context
			ctx := r.Context()
			ctx = context.WithValue(ctx, UserKey, userCtx)
			ctx = context.WithValue(ctx, TokenKey, accessToken)

			// 调用下一个处理器
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// JSONMiddleware JSON 响应中间件
func JSONMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// writeJSON 写入 JSON 响应
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
