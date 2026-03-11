package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	LLM    LLMConfig
	Server ServerConfig
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
	return nil
}
