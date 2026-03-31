package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	ErrNotFound      = errors.New("resource not found")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrRateLimit     = errors.New("rate limit exceeded")
	ErrAPIError      = errors.New("GitHub API error")
)

// Client GitHub API 客户端
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient 创建 GitHub API 客户端
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.github.com",
	}
}

// doRequest 执行 HTTP 请求
func (c *Client) doRequest(ctx context.Context, accessToken, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

// handleErrorResponse 处理错误响应
func (c *Client) handleErrorResponse(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		// 可能是速率限制
		return ErrRateLimit
	default:
		return fmt.Errorf("%w: status %d", ErrAPIError, resp.StatusCode)
	}
}

// Repository 仓库信息
type Repository struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
	HTMLURL     string `json:"html_url"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
}

// StarredRepo Star 的仓库（包含完整信息）
type StarredRepo struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	HTMLURL     string `json:"html_url"`
	Language    string `json:"language"`
	Stargazers  int    `json:"stargazers_count"`
	Forks       int    `json:"forks_count"`
	Topics      []string `json:"topics"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
}

// FileContent 文件内容
type FileContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	SHA         string `json:"sha"`
	Content     string `json:"content"`
	DownloadURL string `json:"download_url"`
}

// CreateRepoRequest 创建仓库请求
type CreateRepoRequest struct {
	Name        string `json:"name"`
	Private     bool   `json:"private"`
	Description string `json:"description"`
	AutoInit    bool   `json:"auto_init"`
}

// UpdateFileRequest 更新文件请求
type UpdateFileRequest struct {
	Message string `json:"message"`
	Content string `json:"content"`
	SHA     string `json:"sha,omitempty"`
	Branch  string `json:"branch,omitempty"`
}

// GetUserRepos 获取用户仓库列表
func (c *Client) GetUserRepos(ctx context.Context, accessToken string, page, perPage int) ([]Repository, error) {
	if perPage == 0 {
		perPage = 100
	}
	if page == 0 {
		page = 1
	}

	path := fmt.Sprintf("/user/repos?page=%d&per_page=%d&sort=updated", page, perPage)
	resp, err := c.doRequest(ctx, accessToken, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var repos []Repository
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}

	return repos, nil
}

// CreateRepo 创建仓库
func (c *Client) CreateRepo(ctx context.Context, accessToken string, req *CreateRepoRequest) (*Repository, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, accessToken, "POST", "/user/repos", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, c.handleErrorResponse(resp)
	}

	var repo Repository
	if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		return nil, err
	}

	return &repo, nil
}

// GetUserStarred 获取用户 Star 的仓库列表
func (c *Client) GetUserStarred(ctx context.Context, accessToken, username string, page, perPage int) ([]StarredRepo, error) {
	if perPage == 0 {
		perPage = 100
	}
	if page == 0 {
		page = 1
	}

	path := fmt.Sprintf("/users/%s/starred?page=%d&per_page=%d", username, page, perPage)
	resp, err := c.doRequest(ctx, accessToken, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var repos []StarredRepo
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}

	return repos, nil
}

// GetFile 获取文件内容
func (c *Client) GetFile(ctx context.Context, accessToken, owner, repo, path string) (*FileContent, error) {
	url := fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path)
	resp, err := c.doRequest(ctx, accessToken, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var file FileContent
	if err := json.NewDecoder(resp.Body).Decode(&file); err != nil {
		return nil, err
	}

	return &file, nil
}

// GetFileContent 获取文件内容并解码 base64
func (c *Client) GetFileContent(ctx context.Context, accessToken, owner, repo, path string) (string, string, error) {
	file, err := c.GetFile(ctx, accessToken, owner, repo, path)
	if err != nil {
		return "", "", err
	}

	if file.Content == "" {
		return "", file.SHA, nil
	}

	// 解码 base64 内容
	content, err := base64.StdEncoding.DecodeString(file.Content)
	if err != nil {
		return "", "", err
	}

	return string(content), file.SHA, nil
}

// CreateOrUpdateFile 创建或更新文件
func (c *Client) CreateOrUpdateFile(ctx context.Context, accessToken, owner, repo, path string, req *UpdateFileRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path)
	resp, err := c.doRequest(ctx, accessToken, "PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return c.handleErrorResponse(resp)
	}

	return nil
}
