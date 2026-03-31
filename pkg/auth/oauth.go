package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrOAuthFailed      = errors.New("OAuth authentication failed")
	ErrInvalidCode      = errors.New("invalid authorization code")
	ErrTokenExchange    = errors.New("failed to exchange token")
	ErrGetUserInfo      = errors.New("failed to get user info")
)

// OAuthClient GitHub OAuth 客户端
type OAuthClient struct {
	clientID     string
	clientSecret string
	redirectURI  string
	httpClient   *http.Client
}

// NewOAuthClient 创建 OAuth 客户端
func NewOAuthClient(clientID, clientSecret, redirectURI string) *OAuthClient {
	return &OAuthClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GitHubUser GitHub 用户信息
type GitHubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
	Name      string `json:"name"`
	Email     string `json:"email"`
}

// OAuthToken OAuth token 响应
type OAuthToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// GetAuthURL 生成 OAuth 授权 URL
func (c *OAuthClient) GetAuthURL(state string) string {
	params := url.Values{
		"client_id":    {c.clientID},
		"redirect_uri": {c.redirectURI},
		"scope":        {"repo,read:user"},
		"state":        {state},
	}

	return "https://github.com/login/oauth/authorize?" + params.Encode()
}

// ExchangeCode 使用 code 交换 access token
func (c *OAuthClient) ExchangeCode(ctx context.Context, code string) (*OAuthToken, error) {
	if code == "" {
		return nil, ErrInvalidCode
	}

	// 构造请求
	data := url.Values{
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"code":          {code},
		"redirect_uri":  {c.redirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 解析响应
	var token OAuthToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}

	if token.AccessToken == "" {
		return nil, ErrTokenExchange
	}

	return &token, nil
}

// GetUserInfo 获取 GitHub 用户信息
func (c *OAuthClient) GetUserInfo(ctx context.Context, accessToken string) (*GitHubUser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var user GitHubUser
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, err
	}

	if user.ID == 0 || user.Login == "" {
		return nil, ErrGetUserInfo
	}

	return &user, nil
}

// Authenticate 完整的 OAuth 认证流程（交换 code + 获取用户信息）
func (c *OAuthClient) Authenticate(ctx context.Context, code string) (*GitHubUser, string, error) {
	// 1. 交换 code 获取 access token
	token, err := c.ExchangeCode(ctx, code)
	if err != nil {
		return nil, "", fmt.Errorf("exchange code failed: %w", err)
	}

	// 2. 获取用户信息
	user, err := c.GetUserInfo(ctx, token.AccessToken)
	if err != nil {
		return nil, "", fmt.Errorf("get user info failed: %w", err)
	}

	return user, token.AccessToken, nil
}
