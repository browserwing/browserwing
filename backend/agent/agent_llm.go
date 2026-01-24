package agent

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
	"github.com/Ingenimax/agent-sdk-go/pkg/llm/anthropic"
	"github.com/Ingenimax/agent-sdk-go/pkg/llm/openai"
	"github.com/browserwing/browserwing/models"
)

// LLMAdapter LLM 适配器,用于将各种 LLM 提供商适配为 agent-sdk-go 的接口
type LLMAdapter struct {
	provider string
	model    string
	client   interfaces.LLM
}

// DeepSeekWrapper DeepSeek API 的包装器
// 用于处理 DeepSeek 的特殊参数要求
type DeepSeekWrapper struct {
	client interfaces.LLM
}

// DeepSeekToolWrapper 包装工具以确保工具名称符合 DeepSeek 的要求
type DeepSeekToolWrapper struct {
	tool         interfaces.Tool
	originalName string
	cleanedName  string
}

// cleanToolName 清理工具名称，使其符合 DeepSeek 的要求 ^[a-zA-Z0-9_-]+$
func cleanToolName(name string) string {
	// 将不允许的字符替换为下划线
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	cleaned := re.ReplaceAllString(name, "_")

	// 确保不以数字或特殊字符开头
	if len(cleaned) > 0 && !regexp.MustCompile(`^[a-zA-Z]`).MatchString(cleaned) {
		cleaned = "tool_" + cleaned
	}

	return cleaned
}

// wrapToolsForDeepSeek 包装工具数组，清理工具名称
func (d *DeepSeekWrapper) wrapToolsForDeepSeek(tools []interfaces.Tool) []interfaces.Tool {
	wrappedTools := make([]interfaces.Tool, len(tools))
	for i, tool := range tools {
		originalName := tool.Name()
		cleanedName := cleanToolName(originalName)

		// 如果名称需要清理，则使用包装器
		if cleanedName != originalName {
			wrappedTools[i] = &DeepSeekToolWrapper{
				tool:         tool,
				originalName: originalName,
				cleanedName:  cleanedName,
			}
		} else {
			wrappedTools[i] = tool
		}
	}
	return wrappedTools
}

// Name 实现 interfaces.Tool 接口
func (w *DeepSeekToolWrapper) Name() string {
	return w.cleanedName
}

// Description 实现 interfaces.Tool 接口
func (w *DeepSeekToolWrapper) Description() string {
	return w.tool.Description()
}

// Parameters 实现 interfaces.Tool 接口
func (w *DeepSeekToolWrapper) Parameters() map[string]interfaces.ParameterSpec {
	return w.tool.Parameters()
}

// Run 实现 interfaces.Tool 接口
func (w *DeepSeekToolWrapper) Run(ctx context.Context, input string) (string, error) {
	return w.tool.Run(ctx, input)
}

// Execute 实现 interfaces.Tool 接口
func (w *DeepSeekToolWrapper) Execute(ctx context.Context, args string) (string, error) {
	return w.tool.Execute(ctx, args)
}

// Generate 实现 interfaces.LLM 接口
func (d *DeepSeekWrapper) Generate(ctx context.Context, prompt string, options ...interfaces.GenerateOption) (string, error) {
	// 修正选项中的 top_p 参数
	fixedOptions := d.fixTopPOptions(options)
	return d.client.Generate(ctx, prompt, fixedOptions...)
}

// GenerateWithTools 实现 interfaces.LLM 接口
func (d *DeepSeekWrapper) GenerateWithTools(ctx context.Context, prompt string, tools []interfaces.Tool, options ...interfaces.GenerateOption) (string, error) {
	// 清理工具名称
	wrappedTools := d.wrapToolsForDeepSeek(tools)
	// 修正选项中的 top_p 参数
	fixedOptions := d.fixTopPOptions(options)
	return d.client.GenerateWithTools(ctx, prompt, wrappedTools, fixedOptions...)
}

// GenerateDetailed 实现 interfaces.LLM 接口
func (d *DeepSeekWrapper) GenerateDetailed(ctx context.Context, prompt string, options ...interfaces.GenerateOption) (*interfaces.LLMResponse, error) {
	// 修正选项中的 top_p 参数
	fixedOptions := d.fixTopPOptions(options)
	return d.client.GenerateDetailed(ctx, prompt, fixedOptions...)
}

// GenerateWithToolsDetailed 实现 interfaces.LLM 接口
func (d *DeepSeekWrapper) GenerateWithToolsDetailed(ctx context.Context, prompt string, tools []interfaces.Tool, options ...interfaces.GenerateOption) (*interfaces.LLMResponse, error) {
	// 清理工具名称
	wrappedTools := d.wrapToolsForDeepSeek(tools)
	// 修正选项中的 top_p 参数
	fixedOptions := d.fixTopPOptions(options)
	return d.client.GenerateWithToolsDetailed(ctx, prompt, wrappedTools, fixedOptions...)
}

// Name 实现 interfaces.LLM 接口
func (d *DeepSeekWrapper) Name() string {
	return d.client.Name()
}

// SupportsStreaming 实现 interfaces.LLM 接口
func (d *DeepSeekWrapper) SupportsStreaming() bool {
	return d.client.SupportsStreaming()
}

// GenerateStream 实现 interfaces.StreamingLLM 接口
func (d *DeepSeekWrapper) GenerateStream(ctx context.Context, prompt string, options ...interfaces.GenerateOption) (<-chan interfaces.StreamEvent, error) {
	// 检查底层客户端是否支持流式
	streamingLLM, ok := d.client.(interfaces.StreamingLLM)
	if !ok {
		return nil, fmt.Errorf("underlying client does not support streaming")
	}

	// 修正选项中的 top_p 参数
	fixedOptions := d.fixTopPOptions(options)
	return streamingLLM.GenerateStream(ctx, prompt, fixedOptions...)
}

// GenerateWithToolsStream 实现 interfaces.StreamingLLM 接口
func (d *DeepSeekWrapper) GenerateWithToolsStream(ctx context.Context, prompt string, tools []interfaces.Tool, options ...interfaces.GenerateOption) (<-chan interfaces.StreamEvent, error) {
	// 检查底层客户端是否支持流式
	streamingLLM, ok := d.client.(interfaces.StreamingLLM)
	if !ok {
		return nil, fmt.Errorf("underlying client does not support streaming")
	}

	// 清理工具名称
	wrappedTools := d.wrapToolsForDeepSeek(tools)
	// 修正选项中的 top_p 参数
	fixedOptions := d.fixTopPOptions(options)
	return streamingLLM.GenerateWithToolsStream(ctx, prompt, wrappedTools, fixedOptions...)
}

// fixTopPOptions 修正 top_p 参数，确保在 (0, 1] 范围内
func (d *DeepSeekWrapper) fixTopPOptions(options []interfaces.GenerateOption) []interfaces.GenerateOption {
	// 应用选项到配置以读取当前值
	opts := &interfaces.GenerateOptions{
		LLMConfig: &interfaces.LLMConfig{},
	}
	for _, opt := range options {
		opt(opts)
	}

	// DeepSeek 特殊处理：top_p 必须在 (0, 1] 范围内
	// 如果 top_p 为 0 或 1，调整为 0.95
	if opts.LLMConfig.TopP <= 0 || opts.LLMConfig.TopP >= 1 {
		opts.LLMConfig.TopP = 0.95
	}

	// 重新构建选项
	newOptions := []interfaces.GenerateOption{
		interfaces.WithTemperature(opts.LLMConfig.Temperature),
		interfaces.WithTopP(opts.LLMConfig.TopP),
		interfaces.WithFrequencyPenalty(opts.LLMConfig.FrequencyPenalty),
		interfaces.WithPresencePenalty(opts.LLMConfig.PresencePenalty),
	}

	if len(opts.LLMConfig.StopSequences) > 0 {
		newOptions = append(newOptions, interfaces.WithStopSequences(opts.LLMConfig.StopSequences))
	}

	// 复制其他选项
	if opts.SystemMessage != "" {
		newOptions = append(newOptions, func(o *interfaces.GenerateOptions) {
			o.SystemMessage = opts.SystemMessage
		})
	}
	if opts.ResponseFormat != nil {
		newOptions = append(newOptions, func(o *interfaces.GenerateOptions) {
			o.ResponseFormat = opts.ResponseFormat
		})
	}
	if opts.MaxIterations > 0 {
		newOptions = append(newOptions, func(o *interfaces.GenerateOptions) {
			o.MaxIterations = opts.MaxIterations
		})
	}
	if opts.Memory != nil {
		newOptions = append(newOptions, func(o *interfaces.GenerateOptions) {
			o.Memory = opts.Memory
		})
	}
	if opts.OrgID != "" {
		newOptions = append(newOptions, func(o *interfaces.GenerateOptions) {
			o.OrgID = opts.OrgID
		})
	}

	return newOptions
}

// CreateLLMClient 根据配置创建 LLM 客户端
// 支持多种 LLM 提供商,包括 OpenAI 兼容的 API
func CreateLLMClient(config *models.LLMConfigModel) (interfaces.LLM, error) {
	provider := strings.ToLower(config.Provider)

	// Anthropic Claude 系列 - 使用原生 SDK
	if provider == "anthropic" || provider == "claude" {
		return createAnthropicClient(config)
	}

	// DeepSeek 需要特殊处理（top_p 不能为 0 或 1）
	if provider == "deepseek" {
		return createDeepSeekClient(config)
	}

	// 其他所有提供商都使用 OpenAI 兼容模式
	return createOpenAICompatibleClient(config)
}

// createAnthropicClient 创建 Anthropic 客户端
func createAnthropicClient(config *models.LLMConfigModel) (interfaces.LLM, error) {
	opts := []anthropic.Option{
		anthropic.WithModel(config.Model),
	}

	// Anthropic 支持自定义 BaseURL (如 Claude 代理)
	if config.BaseURL != "" {
		opts = append(opts, anthropic.WithBaseURL(config.BaseURL))
	}

	client := anthropic.NewClient(config.APIKey, opts...)
	return client, nil
}

// createDeepSeekClient 创建 DeepSeek 客户端
// DeepSeek API 有特殊要求：
// 1. top_p 必须在 (0, 1.0] 区间，不能为 0 或 1（通过包装器处理）
// 2. 工具名称只支持 ^[a-zA-Z0-9_-]+$ 模式
// 3. BaseURL 固定为 https://api.deepseek.com/v1
func createDeepSeekClient(config *models.LLMConfigModel) (interfaces.LLM, error) {
	opts := []openai.Option{
		openai.WithModel(config.Model),
	}

	// DeepSeek 的 BaseURL
	baseURL := "https://api.deepseek.com/v1"
	if config.BaseURL != "" {
		baseURL = config.BaseURL
	}
	opts = append(opts, openai.WithBaseURL(baseURL))

	client := openai.NewClient(config.APIKey, opts...)

	// 使用包装器处理 DeepSeek 的特殊参数要求
	return &DeepSeekWrapper{client: client}, nil
}

// createOpenAICompatibleClient 创建 OpenAI 兼容客户端
// 支持所有 OpenAI API 兼容的服务
func createOpenAICompatibleClient(config *models.LLMConfigModel) (interfaces.LLM, error) {
	provider := strings.ToLower(config.Provider)

	opts := []openai.Option{
		openai.WithModel(config.Model),
	}

	// 根据不同提供商设置 BaseURL
	baseURL := getProviderBaseURL(provider, config.BaseURL)
	if baseURL != "" {
		opts = append(opts, openai.WithBaseURL(baseURL))
	}

	// Ollama 本地运行时不需要真实的 API Key，提供默认值
	apiKey := config.APIKey
	if provider == "ollama" && apiKey == "" {
		apiKey = "ollama" // Ollama 本地不验证 API Key，提供占位符即可
	}

	client := openai.NewClient(apiKey, opts...)
	return client, nil
}

// getProviderBaseURL 获取各提供商的默认 BaseURL
func getProviderBaseURL(provider, customBaseURL string) string {
	// 如果用户自定义了 BaseURL,优先使用
	if customBaseURL != "" {
		return customBaseURL
	}

	// 各提供商的默认 API 端点
	baseURLMap := map[string]string{
		// 国际模型
		"openai":     "https://api.openai.com/v1",
		"gemini":     "https://generativelanguage.googleapis.com/v1beta/openai",
		"mistral":    "https://api.mistral.ai/v1",
		"deepseek":   "https://api.deepseek.com",
		"groq":       "https://api.groq.com/openai/v1",
		"cohere":     "https://api.cohere.ai/v1",
		"xai":        "https://api.x.ai/v1",
		"together":   "https://api.together.xyz/v1",
		"novita":     "https://api.novita.ai/v3/openai",
		"openrouter": "https://openrouter.ai/api/v1",

		// 国内模型
		"qwen":        "https://dashscope.aliyuncs.com/compatible-mode/v1",
		"siliconflow": "https://api.siliconflow.cn/v1",
		"doubao":      "https://ark.cn-beijing.volces.com/api/v3",
		"ernie":       "https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop",
		"spark":       "https://spark-api-open.xf-yun.com/v1",
		"chatglm":     "https://open.bigmodel.cn/api/paas/v4",
		"360":         "https://api.360.cn/v1",
		"hunyuan":     "https://hunyuan.tencentcloudapi.com",
		"moonshot":    "https://api.moonshot.cn/v1",
		"baichuan":    "https://api.baichuan-ai.com/v1",
		"minimax":     "https://api.minimax.chat/v1",
		"yi":          "https://api.lingyiwanwu.com/v1",
		"stepfun":     "https://api.stepfun.com/v1",
		"coze":        "https://api.coze.cn/open_api/v2",

		// 本地模型
		"ollama": "http://localhost:11434/v1",
	}

	if url, ok := baseURLMap[provider]; ok {
		return url
	}

	// 未知提供商,返回空字符串,使用 OpenAI SDK 默认值
	return ""
}

// GetProviderInfo 获取提供商信息 (用于日志和调试)
func GetProviderInfo(config *models.LLMConfigModel) string {
	provider := strings.ToLower(config.Provider)
	baseURL := getProviderBaseURL(provider, config.BaseURL)

	info := fmt.Sprintf("%s (%s)", config.Provider, config.Model)
	if baseURL != "" {
		info += fmt.Sprintf(" @ %s", baseURL)
	}

	return info
}

// ValidateLLMConfig 验证 LLM 配置
func ValidateLLMConfig(config *models.LLMConfigModel) error {
	if config.Provider == "" {
		return fmt.Errorf("provider cannot be empty")
	}

	// Ollama 本地运行时不需要 API Key
	provider := strings.ToLower(config.Provider)
	if provider != "ollama" && config.APIKey == "" {
		return fmt.Errorf("api_key cannot be empty")
	}

	if config.Model == "" {
		return fmt.Errorf("model cannot be empty")
	}

	return nil
}

// GetRecommendedModels 获取各提供商的推荐模型列表
func GetRecommendedModels(provider string) []string {
	provider = strings.ToLower(provider)

	modelsMap := map[string][]string{
		// 国际模型
		"openai": {
			"gpt-4o",
			"gpt-4o-mini",
			"gpt-4-turbo",
			"gpt-4",
			"gpt-3.5-turbo",
		},
		"anthropic": {
			"claude-3-5-sonnet-20241022",
			"claude-3-5-haiku-20241022",
			"claude-3-opus-20240229",
			"claude-3-sonnet-20240229",
			"claude-3-haiku-20240307",
		},
		"claude": { // anthropic 别名
			"claude-3-5-sonnet-20241022",
			"claude-3-5-haiku-20241022",
			"claude-3-opus-20240229",
		},
		"gemini": {
			"gemini-2.0-flash-exp",
			"gemini-1.5-pro",
			"gemini-1.5-flash",
		},
		"mistral": {
			"mistral-large-latest",
			"mistral-medium-latest",
			"mistral-small-latest",
		},
		"deepseek": {
			"deepseek-chat",
			"deepseek-coder",
		},
		"groq": {
			"llama-3.3-70b-versatile",
			"llama-3.1-70b-versatile",
			"mixtral-8x7b-32768",
		},
		"xai": {
			"grok-beta",
			"grok-vision-beta",
		},

		// 国内模型
		"qwen": {
			"qwen-max",
			"qwen-plus",
			"qwen-turbo",
			"qwen-long",
		},
		"siliconflow": {
			"deepseek-ai/DeepSeek-V3",
			"Qwen/Qwen2.5-72B-Instruct",
			"meta-llama/Llama-3.3-70B-Instruct",
		},
		"doubao": {
			"doubao-pro-32k",
			"doubao-lite-32k",
		},
		"chatglm": {
			"glm-4-plus",
			"glm-4-air",
			"glm-4-flash",
		},
		"moonshot": {
			"moonshot-v1-8k",
			"moonshot-v1-32k",
			"moonshot-v1-128k",
		},
		"yi": {
			"yi-lightning",
			"yi-large",
			"yi-medium",
		},
		"stepfun": {
			"step-1-8k",
			"step-1-32k",
			"step-1-128k",
		},

		// 本地模型
		"ollama": {
			"qwen2.5:latest",
			"llama3.3:latest",
			"deepseek-r1:latest",
			"mistral:latest",
		},
	}

	if models, ok := modelsMap[provider]; ok {
		return models
	}

	return []string{}
}

// SupportsToolCalling 检查模型是否支持工具调用
func SupportsToolCalling(provider, model string) bool {
	provider = strings.ToLower(provider)
	model = strings.ToLower(model)

	// 已知支持工具调用的模型
	supportedModels := map[string][]string{
		"openai": {
			"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-4",
			"gpt-3.5-turbo", "gpt-3.5-turbo-0125",
		},
		"anthropic": {
			"claude-3-5-sonnet", "claude-3-5-haiku",
			"claude-3-opus", "claude-3-sonnet", "claude-3-haiku",
		},
		"claude": {
			"claude-3-5-sonnet", "claude-3-5-haiku",
			"claude-3-opus", "claude-3-sonnet",
		},
		"gemini": {
			"gemini-2.0", "gemini-1.5-pro", "gemini-1.5-flash",
		},
		"mistral": {
			"mistral-large", "mistral-medium", "mistral-small",
		},
		"deepseek": {
			"deepseek-chat", "deepseek-reasoner", "deepseek-v3",
		},
		"qwen": {
			"qwen-max", "qwen-plus", "qwen-turbo", "qwen2.5",
		},
		"chatglm": {
			"glm-4-plus", "glm-4-air", "glm-4",
		},
		"moonshot": {
			"moonshot-v1",
		},
		"yi": {
			"yi-lightning", "yi-large",
		},
		"siliconflow": {
			"qwen2.5", "qwen/qwen", "deepseek-v3", "deepseek-ai/deepseek",
			"llama-3.3", "llama-3.1", "meta-llama/llama",
			"yi-lightning", "01-ai/yi",
		},
		"ollama": {
			// Ollama 支持工具调用的模型（需要较新的模型）
			"qwen2.5", "qwen2", "qwen",
			"llama3.3", "llama3.2", "llama3.1", "llama3",
			"llama-3.3", "llama-3.2", "llama-3.1", "llama-3",
			"mistral", "mixtral",
			"deepseek-r1", "deepseek-v3", "deepseek-coder",
			"yi-coder", "yi-lightning",
			"phi3", "phi4",
			"gemma2", "gemma",
			"command-r", "command-r-plus",
		},
	}

	// 检查提供商
	models, exists := supportedModels[provider]
	if !exists {
		// 未知提供商，默认支持（OpenAI 兼容）
		return true
	}

	// 检查模型名称是否包含支持的关键词
	for _, supportedModel := range models {
		if strings.Contains(model, supportedModel) {
			return true
		}
	}

	return false
}
