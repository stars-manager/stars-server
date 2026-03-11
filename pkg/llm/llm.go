package llm

import (
	"context"
	"errors"

	"github.com/sashabaranov/go-openai"
	"server/pkg/config"
)

type Client struct {
	client *openai.Client
	config *config.LLMConfig
}

func NewClient(cfg *config.LLMConfig) *Client {
	c := openai.DefaultConfig(cfg.APIKey)
	c.BaseURL = cfg.BaseURL
	c.HTTPClient.Timeout = cfg.Timeout
	return &Client{client: openai.NewClientWithConfig(c), config: cfg}
}

func (c *Client) ChatCompletion(ctx context.Context, messages []openai.ChatCompletionMessage, opts ...Option) (string, error) {
	o := &Options{Temperature: c.config.Temperature, MaxTokens: c.config.MaxTokens}
	for _, opt := range opts {
		opt(o)
	}

	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       c.config.ModelName,
		Messages:    messages,
		Temperature: o.Temperature,
		MaxTokens:   o.MaxTokens,
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("no response")
	}
	return resp.Choices[0].Message.Content, nil
}

func (c *Client) ChatCompletionWithSystem(ctx context.Context, system string, messages []openai.ChatCompletionMessage, opts ...Option) (string, error) {
	msgs := append([]openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleSystem, Content: system}}, messages...)
	return c.ChatCompletion(ctx, msgs, opts...)
}

type Options struct {
	Temperature float32
	MaxTokens   int
}

type Option func(*Options)

func WithTemperature(t float32) Option { return func(o *Options) { o.Temperature = t } }
func WithMaxTokens(m int) Option       { return func(o *Options) { o.MaxTokens = m } }
