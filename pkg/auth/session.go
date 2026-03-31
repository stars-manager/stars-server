package auth

import (
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	CookieName = "github_star_manager_session"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrTokenExpired     = errors.New("token expired")
	ErrMissingClaims    = errors.New("missing required claims")
)

// Claims JWT claims 结构体
type Claims struct {
	UserID         string `json:"user_id"`
	Username       string `json:"username"`
	AvatarURL      string `json:"avatar_url"`
	EncryptedToken string `json:"token"` // AES-GCM 加密的 GitHub access token
	jwt.RegisteredClaims
}

// SessionManager Session 管理器
type SessionManager struct {
	secret        []byte
	encryptionKey []byte
	maxAge        time.Duration
	secure        bool
}

// NewSessionManager 创建 Session 管理器
func NewSessionManager(secret, encryptionKey string, maxAge time.Duration, secure bool) (*SessionManager, error) {
	// Base64 解码 secret
	secretBytes, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return nil, errors.New("secret must be base64 encoded")
	}
	if len(secretBytes) < 32 {
		return nil, errors.New("secret must be at least 32 bytes after base64 decoding")
	}

	// Base64 解码 encryption key
	encryptionKeyBytes, err := base64.StdEncoding.DecodeString(encryptionKey)
	if err != nil {
		return nil, errors.New("encryption key must be base64 encoded")
	}
	if len(encryptionKeyBytes) != 32 {
		return nil, errors.New("encryption key must be exactly 32 bytes after base64 decoding")
	}

	return &SessionManager{
		secret:        secretBytes,
		encryptionKey: encryptionKeyBytes,
		maxAge:        maxAge,
		secure:        secure,
	}, nil
}

// CreateSession 创建新的 session（生成 JWT 并设置 Cookie）
func (sm *SessionManager) CreateSession(userID, username, avatarURL, accessToken string) (string, error) {
	// 加密 access token
	encryptedToken, err := EncryptToken(accessToken, sm.encryptionKey)
	if err != nil {
		return "", err
	}

	// 创建 JWT claims
	now := time.Now()
	claims := &Claims{
		UserID:         userID,
		Username:       username,
		AvatarURL:      avatarURL,
		EncryptedToken: encryptedToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(sm.maxAge)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	// 生成 JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(sm.secret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ParseSession 解析并验证 session（JWT）
func (sm *SessionManager) ParseSession(tokenString string) (*Claims, error) {
	// 解析 JWT
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return sm.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	// 验证必要的 claims
	if claims.UserID == "" || claims.Username == "" || claims.EncryptedToken == "" {
		return nil, ErrMissingClaims
	}

	return claims, nil
}

// GetAccessToken 从 claims 中解密并返回 access token
func (sm *SessionManager) GetAccessToken(claims *Claims) (string, error) {
	return DecryptToken(claims.EncryptedToken, sm.encryptionKey)
}

// SetCookie 设置 session cookie
func (sm *SessionManager) SetCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   sm.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sm.maxAge.Seconds()),
	})
}

// ClearCookie 清除 session cookie
func (sm *SessionManager) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   sm.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1, // 立即过期
	})
}

// GetSessionFromRequest 从 HTTP 请求中获取 session
func (sm *SessionManager) GetSessionFromRequest(r *http.Request) (*Claims, error) {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}

	return sm.ParseSession(cookie.Value)
}
