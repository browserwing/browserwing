package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/browserwing/browserwing/config"
	"github.com/browserwing/browserwing/models"
	"github.com/browserwing/browserwing/storage"
	"github.com/gotoailab/llmhub"
)

// Extractor LLM提取器，用于生成数据提取的JS代码
type Extractor struct {
	llmClient *llmhub.Client
	config    *config.LLMConfig
	storage   *storage.BoltDB
}

// ExtractionRequest 提取请求
type ExtractionRequest struct {
	HTML        string `json:"html"`        // 要提取数据的HTML
	Description string `json:"description"` // 用户对提取内容的描述（可选）
}

// ExtractionResult 提取结果
type ExtractionResult struct {
	JavaScript string `json:"javascript"` // 生成的JS代码
	UsedModel  string `json:"used_model"` // 使用的模型
}

// FormFillRequest 表单填充请求
type FormFillRequest struct {
	HTML        string `json:"html"`        // 表单HTML
	Description string `json:"description"` // 用户对填充内容的描述
}

// FormFillResult 表单填充结果
type FormFillResult struct {
	JavaScript string `json:"javascript"` // 生成的表单填充JS代码
	UsedModel  string `json:"used_model"` // 使用的模型
}

// NewExtractor 创建提取器
func NewExtractor(cfg *config.LLMConfig, db *storage.BoltDB) (*Extractor, error) {
	client, err := llmhub.NewClient(llmhub.ClientConfig{
		APIKey:   cfg.APIKey,
		Provider: llmhub.Provider(cfg.Provider),
		Model:    cfg.Model,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	return &Extractor{
		llmClient: client,
		config:    cfg,
		storage:   db,
	}, nil
}

// GenerateExtractionJS 生成提取数据的JavaScript代码
func (e *Extractor) GenerateExtractionJS(ctx context.Context, req ExtractionRequest) (*ExtractionResult, error) {
	prompt := e.buildPrompt(req)

	// 调用LLM生成代码
	resp, err := e.llmClient.ChatCompletions(context.Background(), llmhub.ChatCompletionRequest{
		Messages: []llmhub.ChatMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: floatPtr(0.3), // 使用较低的温度以获得更确定的代码
		MaxTokens:   intPtr(2000),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("LLM did not return any results")
	}

	// 获取生成的内容
	contentInterface := resp.Choices[0].Message.Content
	content, ok := contentInterface.(string)
	if !ok {
		return nil, fmt.Errorf("failed to parse LLM response content")
	}

	// 提取生成的JavaScript代码
	jsCode := e.extractJavaScript(content)

	return &ExtractionResult{
		JavaScript: jsCode,
		UsedModel:  e.config.Model,
	}, nil
}

// 辅助函数
func floatPtr(f float64) *float64 {
	return &f
}

func intPtr(i int) *int {
	return &i
}

// buildPrompt 构建提示词
func (e *Extractor) buildPrompt(req ExtractionRequest) string {
	var sb strings.Builder

	// 从数据库获取系统提示词
	systemPrompt, err := e.storage.GetPrompt(models.SystemPromptExtractorID)
	if err != nil || systemPrompt == nil {
		// 如果获取失败，使用默认提示词
		sb.WriteString("你是一个专业的数据提取专家。请分析下面的HTML代码，生成一个JavaScript函数来提取结构化数据。\n\n")
	} else {
		sb.WriteString(systemPrompt.Content)
		sb.WriteString("\n\n")
	}

	if req.Description != "" {
		sb.WriteString(fmt.Sprintf("用户需求：%s\n\n", req.Description))
	}

	sb.WriteString("HTML代码：\n```html\n")
	sb.WriteString(req.HTML)
	sb.WriteString("\n```\n\n")

	sb.WriteString("现在请生成代码：")

	return sb.String()
}

// extractJavaScript 从LLM响应中提取JavaScript代码
func (e *Extractor) extractJavaScript(content string) string {
	// 移除markdown代码块标记
	content = strings.TrimSpace(content)

	// 查找 ```javascript 或 ```js 代码块
	if idx := strings.Index(content, "```javascript"); idx >= 0 {
		content = content[idx+13:]
		if endIdx := strings.Index(content, "```"); endIdx >= 0 {
			content = content[:endIdx]
		}
	} else if idx := strings.Index(content, "```js"); idx >= 0 {
		content = content[idx+5:]
		if endIdx := strings.Index(content, "```"); endIdx >= 0 {
			content = content[:endIdx]
		}
	} else if idx := strings.Index(content, "```"); idx >= 0 {
		// 通用代码块
		content = content[idx+3:]
		if endIdx := strings.Index(content, "```"); endIdx >= 0 {
			content = content[:endIdx]
		}
	}

	return strings.TrimSpace(content)
}

// GenerateFormFillJS 生成填充表单的JavaScript代码
func (e *Extractor) GenerateFormFillJS(ctx context.Context, req FormFillRequest) (*FormFillResult, error) {
	prompt := e.buildFormFillPrompt(req)

	// 调用LLM生成代码
	resp, err := e.llmClient.ChatCompletions(context.Background(), llmhub.ChatCompletionRequest{
		Messages: []llmhub.ChatMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: floatPtr(0.3),
		MaxTokens:   intPtr(2000),
	})
	if err != nil {
		return nil, fmt.Errorf("call LLM failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("LLM did not return any results")
	}

	// 获取生成的内容
	contentInterface := resp.Choices[0].Message.Content
	content, ok := contentInterface.(string)
	if !ok {
		return nil, fmt.Errorf("failed to parse LLM response content")
	}

	// 提取生成的JavaScript代码
	jsCode := e.extractJavaScript(content)

	return &FormFillResult{
		JavaScript: jsCode,
		UsedModel:  e.config.Model,
	}, nil
}

// buildFormFillPrompt 构建表单填充提示词
func (e *Extractor) buildFormFillPrompt(req FormFillRequest) string {
	var sb strings.Builder

	// 从数据库获取系统提示词
	systemPrompt, err := e.storage.GetPrompt(models.SystemPromptFormFillerID)
	if err != nil || systemPrompt == nil {
		// 如果获取失败，使用默认提示词
		sb.WriteString("你是一个专业的表单填充专家。请分析下面的HTML表单代码，生成一个JavaScript函数来自动填充表单。\n\n")
		sb.WriteString("要求：\n")
		sb.WriteString("1. 识别表单中的所有输入字段（input, textarea, select等）\n")
		sb.WriteString("2. 根据字段的name、id、placeholder等属性推断其用途\n")
		sb.WriteString("3. 生成合理的测试数据来填充这些字段\n")
		sb.WriteString("4. 返回纯JavaScript代码，不要包含任何解释\n")
		sb.WriteString("5. 代码应该能够直接在浏览器console中执行\n")
		sb.WriteString("6. 使用document.querySelector或querySelectorAll来定位元素\n")
		sb.WriteString("7. 对于select元素，选择合适的选项\n")
		sb.WriteString("8. 对于checkbox和radio，合理选择\n")
		sb.WriteString("9. 触发必要的事件（如input、change事件）以确保表单验证正常工作\n\n")
	} else {
		sb.WriteString(systemPrompt.Content)
		sb.WriteString("\n\n")
	}

	if req.Description != "" {
		sb.WriteString(fmt.Sprintf("用户需求：%s\n\n", req.Description))
	}

	sb.WriteString("表单HTML代码：\n```html\n")
	sb.WriteString(req.HTML)
	sb.WriteString("\n```\n\n")

	sb.WriteString("现在请生成表单填充的JavaScript代码：")

	return sb.String()
}

// ChatSimple 简单的对话接口，用于生成文本响应
func (e *Extractor) ChatSimple(ctx context.Context, prompt string) (string, error) {
	resp, err := e.llmClient.ChatCompletions(ctx, llmhub.ChatCompletionRequest{
		Messages: []llmhub.ChatMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: floatPtr(0.7),
		MaxTokens:   intPtr(2000),
	})
	if err != nil {
		return "", fmt.Errorf("failed to call LLM: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("LLM did not return any results")
	}

	contentInterface := resp.Choices[0].Message.Content
	content, ok := contentInterface.(string)
	if !ok {
		return "", fmt.Errorf("failed to parse LLM response content")
	}

	return content, nil
}

func (e *Extractor) GetMCPInfo(ctx context.Context, name, description, url, actions string) (string, error) {
	systemPrompt, err := e.storage.GetPrompt(models.SystemPromptGetMCPInfoID)
	if err != nil || systemPrompt == nil {
		return "", fmt.Errorf("system prompt for MCP info not found")
	}
	prompt := fmt.Sprintf(systemPrompt.Content, name, description, url, actions)
	return e.ChatSimple(ctx, prompt)
}
