package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"server/pkg/auth"
	githubclient "server/pkg/github"
	"server/pkg/response"
)

// GitHubHandler GitHub API 代理处理器
type GitHubHandler struct {
	client *githubclient.Client
}

// NewGitHubHandler 创建 GitHub 处理器
func NewGitHubHandler(client *githubclient.Client) *GitHubHandler {
	return &GitHubHandler{
		client: client,
	}
}

// GetUserRepos 获取用户仓库列表
// GET /api/v1/github/user/repos
func (h *GitHubHandler) GetUserRepos(w http.ResponseWriter, r *http.Request) {
	// 从 context 获取 token
	token := auth.GetTokenFromContext(r.Context())
	if token == "" {
		response.Error(w, response.CodeUnauthorized, "access token not found")
		return
	}

	// 解析分页参数
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	// 调用 GitHub API
	repos, err := h.client.GetUserRepos(r.Context(), token, page, perPage)
	if err != nil {
		handleGitHubError(w, err)
		return
	}

	response.Success(w, repos)
}

// CreateRepoRequest 创建仓库请求
type CreateRepoRequest struct {
	Name        string `json:"name"`
	Private     bool   `json:"private"`
	Description string `json:"description"`
}

// CreateRepo 创建仓库
// POST /api/v1/github/user/repos
func (h *GitHubHandler) CreateRepo(w http.ResponseWriter, r *http.Request) {
	token := auth.GetTokenFromContext(r.Context())
	if token == "" {
		response.Error(w, response.CodeUnauthorized, "access token not found")
		return
	}

	// 解析请求
	var req CreateRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, response.CodeInvalidJSON, "invalid request body")
		return
	}

	if req.Name == "" {
		response.Error(w, response.CodeInvalidParam, "repository name is required")
		return
	}

	// 创建仓库
	repo, err := h.client.CreateRepo(r.Context(), token, &githubclient.CreateRepoRequest{
		Name:        req.Name,
		Private:     req.Private,
		Description: req.Description,
		AutoInit:    true,
	})
	if err != nil {
		handleGitHubError(w, err)
		return
	}

	response.Success(w, repo)
}

// GetUserStarred 获取用户 Stars 列表
// GET /api/v1/github/user/starred?username=xxx
func (h *GitHubHandler) GetUserStarred(w http.ResponseWriter, r *http.Request) {
	token := auth.GetTokenFromContext(r.Context())
	if token == "" {
		response.Error(w, response.CodeUnauthorized, "access token not found")
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		// 如果没有提供 username，使用当前登录用户
		user := auth.GetUserFromContext(r.Context())
		if user == nil {
			response.Error(w, response.CodeUnauthorized, "user not found")
			return
		}
		username = user.Username
	}

	// 解析分页参数
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	// 获取 Stars 列表
	repos, err := h.client.GetUserStarred(r.Context(), token, username, page, perPage)
	if err != nil {
		handleGitHubError(w, err)
		return
	}

	response.Success(w, repos)
}

// GetFile 获取文件内容
// GET /api/v1/github/repos/{owner}/{repo}/contents/{path}
func (h *GitHubHandler) GetFile(w http.ResponseWriter, r *http.Request) {
	token := auth.GetTokenFromContext(r.Context())
	if token == "" {
		response.Error(w, response.CodeUnauthorized, "access token not found")
		return
	}

	// 从 URL 中提取参数
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	path := r.PathValue("path")

	if owner == "" || repo == "" || path == "" {
		response.Error(w, response.CodeInvalidParam, "owner, repo and path are required")
		return
	}

	// 获取文件
	file, err := h.client.GetFile(r.Context(), token, owner, repo, path)
	if err != nil {
		handleGitHubError(w, err)
		return
	}

	response.Success(w, file)
}

// FileContentResponse 文件内容响应
type FileContentResponse struct {
	Content string `json:"content"`
	SHA     string `json:"sha"`
}

// GetFileContent 获取文件内容（解码 base64）
// GET /api/v1/github/repos/{owner}/{repo}/contents/{path}/decoded
func (h *GitHubHandler) GetFileContent(w http.ResponseWriter, r *http.Request) {
	token := auth.GetTokenFromContext(r.Context())
	if token == "" {
		response.Error(w, response.CodeUnauthorized, "access token not found")
		return
	}

	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	path := r.PathValue("path")

	if owner == "" || repo == "" || path == "" {
		response.Error(w, response.CodeInvalidParam, "owner, repo and path are required")
		return
	}

	// 获取并解码文件内容
	content, sha, err := h.client.GetFileContent(r.Context(), token, owner, repo, path)
	if err != nil {
		handleGitHubError(w, err)
		return
	}

	response.Success(w, FileContentResponse{
		Content: content,
		SHA:     sha,
	})
}

// UpdateFileRequest 更新文件请求
type UpdateFileRequest struct {
	Message string `json:"message"`
	Content string `json:"content"`
	SHA     string `json:"sha"`
}

// UpdateFile 创建或更新文件
// PUT /api/v1/github/repos/{owner}/{repo}/contents/{path}
func (h *GitHubHandler) UpdateFile(w http.ResponseWriter, r *http.Request) {
	token := auth.GetTokenFromContext(r.Context())
	if token == "" {
		response.Error(w, response.CodeUnauthorized, "access token not found")
		return
	}

	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	path := r.PathValue("path")

	if owner == "" || repo == "" || path == "" {
		response.Error(w, response.CodeInvalidParam, "owner, repo and path are required")
		return
	}

	// 解析请求
	var req UpdateFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, response.CodeInvalidJSON, "invalid request body")
		return
	}

	if req.Message == "" || req.Content == "" {
		response.Error(w, response.CodeInvalidParam, "message and content are required")
		return
	}

	// Base64 编码内容
	encodedContent := base64.StdEncoding.EncodeToString([]byte(req.Content))

	// 创建或更新文件
	err := h.client.CreateOrUpdateFile(r.Context(), token, owner, repo, path, &githubclient.UpdateFileRequest{
		Message: req.Message,
		Content: encodedContent,
		SHA:     req.SHA,
	})
	if err != nil {
		handleGitHubError(w, err)
		return
	}

	response.Success(w, map[string]string{
		"message": "file updated successfully",
	})
}

// handleGitHubError 处理 GitHub API 错误
func handleGitHubError(w http.ResponseWriter, err error) {
	switch err {
	case githubclient.ErrNotFound:
		response.Error(w, response.CodeBadRequest, "resource not found")
	case githubclient.ErrUnauthorized:
		response.Error(w, response.CodeUnauthorized, "unauthorized")
	case githubclient.ErrRateLimit:
		response.Error(w, response.CodeInternalError, "rate limit exceeded")
	default:
		response.Error(w, response.CodeInternalError, fmt.Sprintf("GitHub API error: %v", err))
	}
}
