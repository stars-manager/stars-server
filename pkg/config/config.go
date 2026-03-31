package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"
)

type Config struct {
	LLM     LLMConfig
	Server  ServerConfig
	GitHub  GitHubConfig
	Session SessionConfig
}

type LLMConfig struct {
	BaseURL     string
	APIKey      string
	ModelName   string
	Timeout     time.Duration
	Temperature float32
	MaxTokens   int
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type GitHubConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

type SessionConfig struct {
	Secret        string        // JWT 签名密钥
	EncryptionKey string        // Token 加密密钥
	MaxAge        time.Duration // Session 有效期
	Secure        bool          // Cookie Secure 属性（生产环境应为 true）
}

func Load() *Config {
	return &Config{
		LLM: LLMConfig{
			BaseURL:     env("HUNYUAN_BASE_URL", "https://api.hunyuan.cloud.tencent.com/v1"),
			APIKey:      os.Getenv("HUNYUAN_API_KEY"),
			ModelName:   env("HUNYUAN_MODEL", "hunyuan-lite"),
			Timeout:     5 * time.Minute,
			Temperature: 0.7,
			MaxTokens:   4096,
		},
		Server: ServerConfig{
			Port:         env("PORT", "8080"),
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 5 * time.Minute,
		},
		GitHub: GitHubConfig{
			ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
			ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
			RedirectURI:  os.Getenv("GITHUB_REDIRECT_URI"),
		},
		Session: SessionConfig{
			Secret:        os.Getenv("SESSION_SECRET"),
			EncryptionKey: os.Getenv("ENCRYPTION_KEY"),
			MaxAge:        7 * 24 * time.Hour, // 默认 7 天
			Secure:        os.Getenv("SESSION_SECURE") == "true",
		},
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.LLM.APIKey == "" {
		return fmt.Errorf("HUNYUAN_API_KEY is required")
	}
	if c.LLM.Temperature < 0 || c.LLM.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	if c.LLM.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be positive")
	}
	if c.Server.Port == "" {
		return fmt.Errorf("PORT is required")
	}
	
	// 验证 GitHub OAuth 配置
	if c.GitHub.ClientID == "" {
		return fmt.Errorf("GITHUB_CLIENT_ID is required")
	}
	if c.GitHub.ClientSecret == "" {
		return fmt.Errorf("GITHUB_CLIENT_SECRET is required")
	}
	if c.GitHub.RedirectURI == "" {
		return fmt.Errorf("GITHUB_REDIRECT_URI is required")
	}
	
	// 验证 Session 配置
	if c.Session.Secret == "" {
		return fmt.Errorf("SESSION_SECRET is required")
	}
	secretBytes, err := base64.StdEncoding.DecodeString(c.Session.Secret)
	if err != nil {
		return fmt.Errorf("SESSION_SECRET must be base64 encoded")
	}
	if len(secretBytes) < 32 {
		return fmt.Errorf("SESSION_SECRET must decode to at least 32 bytes")
	}

	if c.Session.EncryptionKey == "" {
		return fmt.Errorf("ENCRYPTION_KEY is required")
	}
	keyBytes, err := base64.StdEncoding.DecodeString(c.Session.EncryptionKey)
	if err != nil {
		return fmt.Errorf("ENCRYPTION_KEY must be base64 encoded")
	}
	if len(keyBytes) != 32 {
		return fmt.Errorf("ENCRYPTION_KEY must decode to exactly 32 bytes for AES-256")
	}

	return nil
}
