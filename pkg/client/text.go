package client

import (
	"context"
	"fmt"
)

// TextClient 文本处理客户端
type TextClient struct {
	client *Client
}

// TagRequest 标签请求
type TagRequest struct {
	Text string `json:"text"`
}

// TagResponse 标签响应
type TagResponse struct {
	Tags string `json:"tags"`
}

// SummaryRequest 总结请求
type SummaryRequest struct {
	Text      string `json:"text"`
	MaxLength int    `json:"max_length,omitempty"` // 总结最大长度(字数)
}

// SummaryResponse 总结响应
type SummaryResponse struct {
	Summary string `json:"summary"`
}

// UnderstandingRequest 理解请求
type UnderstandingRequest struct {
	Text string `json:"text"`
}

// UnderstandingResponse 理解响应
type UnderstandingResponse struct {
	Analysis string `json:"analysis"`
}

// Tag 为文本打标签
// 根据文本内容自动生成相关标签
func (t *TextClient) Tag(ctx context.Context, text string) (*TagResponse, error) {
	req := &TagRequest{Text: text}

	resp, err := t.client.post("/api/v1/text/tag", req)
	if err != nil {
		return nil, err
	}

	tags, ok := resp["tags"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return &TagResponse{Tags: tags}, nil
}

// TagWithContext 为文本打标签并总结内容
// 返回标签和简短的内容总结
func (t *TextClient) TagWithContext(ctx context.Context, text string) (string, string, error) {
	// 1. 获取标签
	tagResp, err := t.Tag(ctx, text)
	if err != nil {
		return "", "", fmt.Errorf("tag failed: %w", err)
	}

	// 2. 获取总结
	summaryResp, err := t.Summary(ctx, text, 50)
	if err != nil {
		return tagResp.Tags, "", fmt.Errorf("summary failed: %w", err)
	}

	return tagResp.Tags, summaryResp.Summary, nil
}

// Summary 总结文本
// 对文本进行智能总结，可指定最大长度
func (t *TextClient) Summary(ctx context.Context, text string, maxLength int) (*SummaryResponse, error) {
	req := &SummaryRequest{
		Text:      text,
		MaxLength: maxLength,
	}

	resp, err := t.client.post("/api/v1/text/summary", req)
	if err != nil {
		return nil, err
	}

	summary, ok := resp["summary"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return &SummaryResponse{Summary: summary}, nil
}

// Understand 深度理解文本
// 分析文本的主题、情感、意图等
func (t *TextClient) Understand(ctx context.Context, text string) (*UnderstandingResponse, error) {
	req := &UnderstandingRequest{Text: text}

	resp, err := t.client.post("/api/v1/text/understand", req)
	if err != nil {
		return nil, err
	}

	analysis, ok := resp["analysis"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return &UnderstandingResponse{Analysis: analysis}, nil
}

// Analyze 综合分析文本
// 一次性获取标签、总结和理解结果
func (t *TextClient) Analyze(ctx context.Context, text string) (tags, summary, analysis string, err error) {
	// 并发调用三个接口提高效率
	type result struct {
		tags      string
		summary   string
		analysis  string
		err       error
	}

	results := make(chan result, 3)

	// 标签
	go func() {
		tagResp, err := t.Tag(ctx, text)
		if err != nil {
			results <- result{err: err}
			return
		}
		results <- result{tags: tagResp.Tags}
	}()

	// 总结
	go func() {
		summaryResp, err := t.Summary(ctx, text, 100)
		if err != nil {
			results <- result{err: err}
			return
		}
		results <- result{summary: summaryResp.Summary}
	}()

	// 理解
	go func() {
		understandResp, err := t.Understand(ctx, text)
		if err != nil {
			results <- result{err: err}
			return
		}
		results <- result{analysis: understandResp.Analysis}
	}()

	// 收集结果
	var tagResult, summaryResult, analysisResult string
	var firstErr error

	for i := 0; i < 3; i++ {
		r := <-results
		if r.err != nil && firstErr == nil {
			firstErr = r.err
		}
		if r.tags != "" {
			tagResult = r.tags
		}
		if r.summary != "" {
			summaryResult = r.summary
		}
		if r.analysis != "" {
			analysisResult = r.analysis
		}
	}

	return tagResult, summaryResult, analysisResult, firstErr
}
