package agent

import (
	"fmt"
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

// CreateLLMClient 根据配置创建 LLM 客户端
// 支持多种 LLM 提供商,包括 OpenAI 兼容的 API
func CreateLLMClient(config *models.LLMConfigModel) (interfaces.LLM, error) {
	provider := strings.ToLower(config.Provider)

	// Anthropic Claude 系列 - 使用原生 SDK
	if provider == "anthropic" || provider == "claude" {
		return createAnthropicClient(config)
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

	client := openai.NewClient(config.APIKey, opts...)
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

	if config.APIKey == "" {
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
