package client

import (
	"context"
	"fmt"
	"strings"
)

// SearchClient 搜索客户端
type SearchClient struct {
	client *Client
}

// SearchRequest 搜索请求
type SearchRequest struct {
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
}

// SearchResponse 搜索响应
type SearchResponse struct {
	Answer string `json:"answer"`
}

// Search 基于文档内容的AI搜索
// 根据提供的问题和文档集合，智能生成答案
func (s *SearchClient) Search(ctx context.Context, query string, documents []string) (*SearchResponse, error) {
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if len(documents) == 0 {
		return nil, fmt.Errorf("documents is required")
	}

	req := &SearchRequest{
		Query:     query,
		Documents: documents,
	}

	resp, err := s.client.post("/api/v1/chat/search", req)
	if err != nil {
		return nil, err
	}

	answer, ok := resp["answer"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return &SearchResponse{Answer: answer}, nil
}

// SearchWithSummary 搜索并总结文档
// 先总结每个文档，再基于总结进行搜索
func (s *SearchClient) SearchWithSummary(ctx context.Context, query string, documents []string, summaryLength int) (*SearchResponse, error) {
	// 1. 先总结每个文档
	textClient := s.client.Text()
	summaries := make([]string, len(documents))

	for i, doc := range documents {
		summary, err := textClient.Summary(ctx, doc, summaryLength)
		if err != nil {
			// 如果总结失败，使用原文的前100字
			if len(doc) > 100 {
				summaries[i] = doc[:100] + "..."
			} else {
				summaries[i] = doc
			}
		} else {
			summaries[i] = summary.Summary
		}
	}

	// 2. 基于总结进行搜索
	return s.Search(ctx, query, summaries)
}

// BatchSearch 批量搜索
// 对多个问题进行批量搜索
func (s *SearchClient) BatchSearch(ctx context.Context, queries []string, documents []string) ([]*SearchResponse, error) {
	results := make([]*SearchResponse, len(queries))
	errChan := make(chan error, len(queries))

	// 并发处理每个查询
	for i, query := range queries {
		go func(idx int, q string) {
			resp, err := s.Search(ctx, q, documents)
			if err != nil {
				errChan <- fmt.Errorf("query %d failed: %w", idx, err)
				return
			}
			results[idx] = resp
			errChan <- nil
		}(i, query)
	}

	// 等待所有查询完成
	for i := 0; i < len(queries); i++ {
		if err := <-errChan; err != nil {
			return nil, err
		}
	}

	return results, nil
}

// SearchAndTag 搜索并打标签
// 搜索后对答案和源文档打标签
func (s *SearchClient) SearchAndTag(ctx context.Context, query string, documents []string) (*SearchResponse, []string, error) {
	// 1. 搜索
	searchResp, err := s.Search(ctx, query, documents)
	if err != nil {
		return nil, nil, err
	}

	// 2. 对答案打标签
	textClient := s.client.Text()
	tagResp, err := textClient.Tag(ctx, searchResp.Answer)
	if err != nil {
		return searchResp, nil, err
	}

	// 3. 解析标签
	tags := strings.Split(tagResp.Tags, ",")
	for i, tag := range tags {
		tags[i] = strings.TrimSpace(tag)
	}

	return searchResp, tags, nil
}

// SearchWithDocuments 智能文档搜索
// 自动处理和搜索文档内容
func (s *SearchClient) SearchWithDocuments(ctx context.Context, query string, documents []string, options ...SearchOption) (*SearchResponse, error) {
	opts := &searchOptions{
		summarizeFirst: false,
		summaryLength:  50,
		maxDocLength:   1000,
	}

	for _, opt := range options {
		opt(opts)
	}

	// 预处理文档
	processedDocs := make([]string, len(documents))
	for i, doc := range documents {
		// 截断过长的文档
		if len(doc) > opts.maxDocLength {
			processedDocs[i] = doc[:opts.maxDocLength] + "..."
		} else {
			processedDocs[i] = doc
		}
	}

	// 如果需要先总结
	if opts.summarizeFirst {
		return s.SearchWithSummary(ctx, query, processedDocs, opts.summaryLength)
	}

	return s.Search(ctx, query, processedDocs)
}

type searchOptions struct {
	summarizeFirst bool
	summaryLength  int
	maxDocLength   int
}

// SearchOption 搜索选项
type SearchOption func(*searchOptions)

// WithSummarize 是否在搜索前先总结文档
func WithSummarize(summarize bool) SearchOption {
	return func(o *searchOptions) {
		o.summarizeFirst = summarize
	}
}

// WithSummaryLength 设置总结长度
func WithSummaryLength(length int) SearchOption {
	return func(o *searchOptions) {
		o.summaryLength = length
	}
}

// WithMaxDocLength 设置文档最大长度
func WithMaxDocLength(length int) SearchOption {
	return func(o *searchOptions) {
		o.maxDocLength = length
	}
}
