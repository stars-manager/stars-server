package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client 统一客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
	text       *TextClient
	search     *SearchClient
	chat       *ChatClient
}

// Config 客户端配置
type Config struct {
	BaseURL    string
	Timeout    time.Duration
	HTTPClient *http.Client
}

// NewClient 创建新的统一客户端
func NewClient(cfg *Config) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: cfg.Timeout,
		}
	}

	client := &Client{
		baseURL:    cfg.BaseURL,
		httpClient: httpClient,
	}

	// 初始化各能力客户端
	client.text = &TextClient{client: client}
	client.search = &SearchClient{client: client}
	client.chat = &ChatClient{client: client}

	return client
}

// Text 获取文本处理客户端
func (c *Client) Text() *TextClient {
	return c.text
}

// Search 获取搜索客户端
func (c *Client) Search() *SearchClient {
	return c.search
}

// Chat 获取对话客户端
func (c *Client) Chat() *ChatClient {
	return c.chat
}

// post 发送POST请求
func (c *Client) post(path string, data any) (map[string]any, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal request data failed: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+path,
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	return result, nil
}

// delete 发送DELETE请求
func (c *Client) delete(path string) error {
	req, err := http.NewRequest("DELETE", c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create delete request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// DefaultClient 默认客户端
var DefaultClient *Client

// Init 初始化默认客户端
func Init(baseURL string) {
	DefaultClient = NewClient(&Config{
		BaseURL: baseURL,
	})
}
