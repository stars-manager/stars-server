package text

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"server/pkg/llm"
	"server/pkg/response"
)

// 常量配置
const (
	MaxProjects        = 20   // 最大项目数量
	MaxNameLength      = 200  // 项目名称最大长度
	MaxDescLength      = 2000 // 项目描述最大长度
	MaxUrlLength       = 500  // URL最大长度
	MaxTopicLength     = 50   // 单个主题最大长度
	MaxTopics          = 10   // 最大主题数量
)

// Service 文本处理服务
type Service struct {
	client *llm.Client
}

// NewService 创建文本处理服务
func NewService(client *llm.Client) *Service {
	return &Service{
		client: client,
	}
}

// StarsTagsRequest 项目标签请求
type StarsTagsRequest struct {
	Projects []ProjectInfo `json:"projects"`
}

// ProjectInfo 项目信息
type ProjectInfo struct {
	Name        string   `json:"name"`                  // 项目名称（必填）
	FullName    string   `json:"full_name,omitempty"`   // 完整名称，如 owner/repo
	Description string   `json:"description,omitempty"` // 项目描述
	Language    string   `json:"language,omitempty"`    // 主要编程语言
	URL         string   `json:"url,omitempty"`         // 项目地址
	Stars       int      `json:"stars,omitempty"`       // Star 数
	Forks       int      `json:"forks,omitempty"`       // Fork 数
	Topics      []string `json:"topics,omitempty"`      // 主题标签
}

// Validate 验证请求
func (r *StarsTagsRequest) Validate() error {
	if len(r.Projects) == 0 {
		return fmt.Errorf("项目列表不能为空")
	}
	if len(r.Projects) > MaxProjects {
		return fmt.Errorf("项目数量不能超过%d个", MaxProjects)
	}
	for i := range r.Projects {
		r.Projects[i].Name = strings.TrimSpace(r.Projects[i].Name)
		if r.Projects[i].Name == "" {
			return fmt.Errorf("第%d个项目名称不能为空", i+1)
		}
		if len(r.Projects[i].Name) > MaxNameLength {
			r.Projects[i].Name = r.Projects[i].Name[:MaxNameLength]
		}
		// 描述超长时自动截断
		if len(r.Projects[i].Description) > MaxDescLength {
			r.Projects[i].Description = r.Projects[i].Description[:MaxDescLength] + "..."
		}
		if len(r.Projects[i].URL) > MaxUrlLength {
			r.Projects[i].URL = r.Projects[i].URL[:MaxUrlLength]
		}
		if len(r.Projects[i].Topics) > MaxTopics {
			r.Projects[i].Topics = r.Projects[i].Topics[:MaxTopics]
		}
		for j, topic := range r.Projects[i].Topics {
			if len(topic) > MaxTopicLength {
				r.Projects[i].Topics[j] = topic[:MaxTopicLength]
			}
		}
	}
	return nil
}

// StarsTagsResponse 项目标签响应
type StarsTagsResponse struct {
	Projects     []ProjectSummary `json:"projects"`      // 每个项目的总结和标签
	ProjectCount int              `json:"project_count"` // 处理的项目数量
	ProcessTime  time.Duration    `json:"process_time"`  // 处理耗时
}

// ProjectSummary 单个项目总结
type ProjectSummary struct {
	Name    string   `json:"name"`    // 项目名称
	Summary string   `json:"summary"` // 项目总结
	Tags    []string `json:"tags"`    // 项目标签
}

// StarsTags 为项目生成总结和标签
func (s *Service) StarsTags(ctx context.Context, req *StarsTagsRequest) (*StarsTagsResponse, error) {
	startTime := time.Now()

	// 验证输入
	if err := req.Validate(); err != nil {
		return nil, response.NewError(response.CodeInvalidParam, err.Error())
	}

	// 构建项目信息文本
	var projectInfo strings.Builder
	for i, p := range req.Projects {
		p.Name = strings.TrimSpace(p.Name)
		p.FullName = strings.TrimSpace(p.FullName)
		p.Description = strings.TrimSpace(p.Description)
		p.Language = strings.TrimSpace(p.Language)
		p.URL = strings.TrimSpace(p.URL)

		fmt.Fprintf(&projectInfo, "【项目%d】\n", i+1)
		fmt.Fprintf(&projectInfo, "名称：%s\n", p.Name)
		if p.FullName != "" {
			fmt.Fprintf(&projectInfo, "全名：%s\n", p.FullName)
		}
		if p.Description != "" {
			fmt.Fprintf(&projectInfo, "描述：%s\n", p.Description)
		}
		if p.Language != "" {
			fmt.Fprintf(&projectInfo, "语言：%s\n", p.Language)
		}
		if p.URL != "" {
			fmt.Fprintf(&projectInfo, "地址：%s\n", p.URL)
		}
		if p.Stars > 0 {
			fmt.Fprintf(&projectInfo, "Star数：%d\n", p.Stars)
		}
		if p.Forks > 0 {
			fmt.Fprintf(&projectInfo, "Fork数：%d\n", p.Forks)
		}
		if len(p.Topics) > 0 {
			fmt.Fprintf(&projectInfo, "主题：%s\n", strings.Join(p.Topics, ", "))
		}
		projectInfo.WriteString("\n")
	}

	projectCount := len(req.Projects)

	prompt := fmt.Sprintf(`你是一个 GitHub 项目分类专家。请为以下 %d 个项目生成简洁的中文总结和准确的中文标签。

项目信息：
%s

【输出要求】

1. 总结（summary）：
   - 使用中文，一句话描述项目的核心功能和技术特点
   - 格式：[项目类型] + 核心功能描述
   - 长度：15-30字，控制在合理范围

2. 标签（tags）：
   - 必须使用中文
   - 数量：2-4个标签
   - 长度：每个标签2-8个字
   - 标签类型参考：
     * 技术栈类：前端框架、后端框架、数据库、机器学习、区块链等
     * 应用领域类：Web开发、移动端、DevOps、数据分析、游戏开发等
     * 功能类型类：UI组件、状态管理、网络请求、测试工具、监控等
     * 语言类：如 Python库、Go工具、Rust应用等

【示例】
输入项目：
- name: "Vue.js"
- description: "渐进式 JavaScript 框架"
- language: "TypeScript"

输出：
{
  "name": "Vue.js",
  "summary": "渐进式JavaScript前端框架，支持响应式数据绑定",
  "tags": ["前端框架", "JavaScript", "UI开发"]
}

【重要规则】
1. projects 数组必须包含所有 %d 个项目，按顺序返回
2. 必须严格按照项目名称（name）一一对应
3. 总结和标签必须使用中文
4. 标签要准确反映项目特点，避免泛泛而谈

请严格按照以下 JSON 格式返回，不要包含 markdown 代码块或其他内容：
{
  "projects": [
    {"name": "项目名称", "summary": "中文项目总结", "tags": ["中文标签1", "中文标签2"]}
  ]
}`, projectCount, projectInfo.String(), projectCount)

	result, err := s.client.ChatCompletion(ctx, []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		},
	}, llm.WithTemperature(0.3), llm.WithMaxTokens(800))

	if err != nil {
		return nil, response.NewError(response.CodeLLMError, fmt.Sprintf("标签生成失败: %v", err))
	}

	// 调试：打印 LLM 原始返回
	log.Printf("[LLM Response] %s", result)

	// 解析JSON响应
	var resp struct {
		Projects []ProjectSummary `json:"projects"`
	}

	// 尝试解析JSON，失败则降级处理
	if err := parseJSONResponse(result, &resp); err != nil {
		log.Printf("[LLM Parse Error] %v", err)
		// 降级处理：为每个项目生成默认值
		defaultProjects := make([]ProjectSummary, projectCount)
		for i, p := range req.Projects {
			defaultProjects[i] = ProjectSummary{
				Name:    p.Name,
				Summary: "无法生成总结",
				Tags:    []string{"未分类"},
			}
		}
		return &StarsTagsResponse{
			Projects:     defaultProjects,
			ProjectCount: projectCount,
			ProcessTime:  time.Since(startTime),
		}, nil
	}

	log.Printf("[LLM Parsed] projects=%d", len(resp.Projects))

	// 清理项目总结和标签
	cleanProjects := make([]ProjectSummary, 0, len(resp.Projects))
	for _, p := range resp.Projects {
		p.Name = strings.TrimSpace(p.Name)
		p.Summary = strings.TrimSpace(p.Summary)

		// 清理标签
		cleanTags := make([]string, 0, len(p.Tags))
		for _, tag := range p.Tags {
			t := strings.TrimSpace(tag)
			charCount := len([]rune(t))
			if t != "" && charCount >= 2 && charCount <= 8 {
				cleanTags = append(cleanTags, t)
			}
		}
		if len(cleanTags) == 0 {
			cleanTags = []string{"未分类"}
		}
		p.Tags = cleanTags

		if p.Name != "" && p.Summary != "" {
			cleanProjects = append(cleanProjects, p)
		}
	}

	log.Printf("[Final] projects=%d", len(cleanProjects))

	return &StarsTagsResponse{
		Projects:     cleanProjects,
		ProjectCount: projectCount,
		ProcessTime:  time.Since(startTime),
	}, nil
}

// parseJSONResponse 从LLM响应中提取JSON
func parseJSONResponse(input string, target any) error {
	// 清理输入
	input = strings.TrimSpace(input)

	// 使用正则移除 markdown 代码块
	// 匹配 ```json ... ``` 或 ``` ... ```
	re := regexp.MustCompile("(?s)^```(?:json)?\\s*\n?(.*?)\n?```$")
	if matches := re.FindStringSubmatch(input); len(matches) > 1 {
		input = strings.TrimSpace(matches[1])
	}

	// 尝试直接解析
	if err := json.Unmarshal([]byte(input), target); err == nil {
		return nil
	}

	// 尝试提取JSON块
	start := strings.Index(input, "{")
	end := strings.LastIndex(input, "}")
	if start != -1 && end > start {
		jsonStr := input[start : end+1]
		return json.Unmarshal([]byte(jsonStr), target)
	}

	return fmt.Errorf("无法从响应中提取JSON")
}
