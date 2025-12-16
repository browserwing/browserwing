# Agent LLM 多模型适配实现

## 概述

本次改进实现了 Agent 与前端 LLM 管理的完全联动,支持超过 20+ 种 LLM 提供商,并实现了配置的热加载功能。

## 新增文件

### `backend/agent/agent_llm.go`

LLM 适配器模块,负责将各种 LLM 提供商适配为 agent-sdk-go 的统一接口。

**核心功能**:
- `CreateLLMClient()` - 根据配置创建 LLM 客户端
- `getProviderBaseURL()` - 自动配置各提供商的 API 端点
- `GetRecommendedModels()` - 获取推荐模型列表
- `ValidateLLMConfig()` - 验证配置有效性

## 支持的 LLM 提供商

### 国际模型
| 提供商 | 默认 API 端点 | 推荐模型 |
|--------|--------------|----------|
| OpenAI | https://api.openai.com/v1 | gpt-4o, gpt-4-turbo, gpt-3.5-turbo |
| Anthropic | - (原生SDK) | claude-3-5-sonnet, claude-3-opus |
| Google Gemini | https://generativelanguage.googleapis.com/v1beta/openai | gemini-2.0-flash-exp, gemini-1.5-pro |
| Mistral | https://api.mistral.ai/v1 | mistral-large-latest |
| DeepSeek | https://api.deepseek.com | deepseek-chat, deepseek-coder |
| Groq | https://api.groq.com/openai/v1 | llama-3.3-70b-versatile |
| Cohere | https://api.cohere.ai/v1 | - |
| xAI | https://api.x.ai/v1 | grok-beta |
| together.ai | https://api.together.xyz/v1 | - |
| novita.ai | https://api.novita.ai/v3/openai | - |
| OpenRouter | https://openrouter.ai/api/v1 | - |

### 国内模型
| 提供商 | 默认 API 端点 | 推荐模型 |
|--------|--------------|----------|
| 通义千问 | https://dashscope.aliyuncs.com/compatible-mode/v1 | qwen-max, qwen-plus |
| 硅基流动 | https://api.siliconflow.cn/v1 | deepseek-ai/DeepSeek-V3 |
| 豆包 | https://ark.cn-beijing.volces.com/api/v3 | doubao-pro-32k |
| 文心一言 | https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop | - |
| 讯飞星火 | https://spark-api-open.xf-yun.com/v1 | - |
| ChatGLM (智谱) | https://open.bigmodel.cn/api/paas/v4 | glm-4-plus, glm-4-flash |
| 360智脑 | https://api.360.cn/v1 | - |
| 腾讯混元 | https://hunyuan.tencentcloudapi.com | - |
| Moonshot AI | https://api.moonshot.cn/v1 | moonshot-v1-8k, moonshot-v1-128k |
| 百川大模型 | https://api.baichuan-ai.com/v1 | - |
| MINIMAX | https://api.minimax.chat/v1 | - |
| 零一万物 | https://api.lingyiwanwu.com/v1 | yi-lightning |
| 阶跃星辰 | https://api.stepfun.com/v1 | step-1-8k, step-1-128k |
| Coze | https://api.coze.cn/open_api/v2 | - |

### 本地模型
| 提供商 | 默认 API 端点 | 推荐模型 |
|--------|--------------|----------|
| Ollama | http://localhost:11434/v1 | qwen2.5, llama3.3, deepseek-r1 |

## 工作原理

### 1. LLM 适配策略

```go
// Anthropic 使用原生 SDK
if provider == "anthropic" || provider == "claude" {
    return anthropic.NewClient(apiKey, anthropic.WithModel(model))
}

// 其他提供商使用 OpenAI 兼容模式
return openai.NewClient(apiKey, 
    openai.WithModel(model),
    openai.WithBaseURL(baseURL))
```

**为什么这样设计?**
- Anthropic 有专用 SDK,提供更好的原生支持
- 其他大部分提供商都兼容 OpenAI API 格式
- 统一接口,降低集成复杂度

### 2. 配置热加载流程

```
前端 LLM 管理页面
    ↓
创建/更新配置 → 保存到数据库
    ↓
Handler.CreateLLMConfig / UpdateLLMConfig
    ↓
检查是否是默认或启用的配置
    ↓
自动调用 AgentManager.ReloadLLM()
    ↓
Agent 使用新的 LLM 配置
```

### 3. 数据流向

```
用户在前端配置 LLM
    ↓
LLMConfigModel 保存到 BoltDB
    ↓
AgentManager 从数据库加载配置
    ↓
agent_llm.CreateLLMClient() 创建客户端
    ↓
ChatSession 使用 LLM 处理对话
```

## 核心代码改动

### 1. `agent.go` 改动

**之前**:
```go
// 手动设置 LLM 提供者
func (am *AgentManager) SetLLMProvider(llmConfig *models.LLMConfigModel) error {
    switch llmConfig.Provider {
    case "openai":
        client = openai.NewClient(...)
    case "anthropic":
        client = anthropic.NewClient(...)
    default:
        return fmt.Errorf("不支持的提供者")
    }
}
```

**现在**:
```go
// 自动从数据库加载
func (am *AgentManager) LoadLLMFromDatabase() error {
    configs, _ := am.db.ListLLMConfigs()
    selectedConfig := findDefaultOrFirstActive(configs)
    return am.SetLLMConfig(selectedConfig)
}

func (am *AgentManager) SetLLMConfig(config *models.LLMConfigModel) error {
    client, err := CreateLLMClient(config) // 使用适配器
    am.llmClient = client
}
```

### 2. `handlers.go` 改动

**新增自动通知机制**:
```go
func (h *Handler) CreateLLMConfig(c *gin.Context) {
    // ... 保存配置 ...
    
    // 如果是默认或启用的配置,通知 Agent 重新加载
    if (req.IsDefault || req.IsActive) && h.agentManager != nil {
        if am, ok := h.agentManager.(interface{ ReloadLLM() error }); ok {
            am.ReloadLLM()
        }
    }
}
```

### 3. 新增 API 端点

- `POST /api/v1/agent/llm/set` - 手动设置 LLM 配置
- `POST /api/v1/agent/llm/reload` - 手动重新加载 LLM 配置

## 使用方式

### 方式 1: 在前端配置(推荐)

1. 访问 http://localhost:8080/llm
2. 点击"添加配置"
3. 选择提供商、填写 API Key 和模型
4. 保存后 Agent 自动加载

### 方式 2: API 调用

```bash
# 创建配置
curl -X POST http://localhost:8080/api/v1/llm-configs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-deepseek",
    "provider": "deepseek",
    "api_key": "sk-xxx",
    "model": "deepseek-chat",
    "is_default": true,
    "is_active": true
  }'

# Agent 会自动重新加载
```

### 方式 3: 手动触发重载

```bash
curl -X POST http://localhost:8080/api/v1/agent/llm/reload
```

## 配置示例

### DeepSeek
```json
{
  "name": "deepseek-chat",
  "provider": "deepseek",
  "api_key": "sk-xxx",
  "model": "deepseek-chat",
  "base_url": "",  // 自动使用 https://api.deepseek.com
  "is_default": true,
  "is_active": true
}
```

### 硅基流动
```json
{
  "name": "siliconflow-deepseek",
  "provider": "siliconflow",
  "api_key": "sk-xxx",
  "model": "deepseek-ai/DeepSeek-V3",
  "base_url": "",  // 自动使用 https://api.siliconflow.cn/v1
  "is_default": true,
  "is_active": true
}
```

### Ollama (本地)
```json
{
  "name": "local-qwen",
  "provider": "ollama",
  "api_key": "ollama",  // 任意值即可
  "model": "qwen2.5:latest",
  "base_url": "http://localhost:11434/v1",
  "is_default": true,
  "is_active": true
}
```

### 自定义 OpenAI 兼容服务
```json
{
  "name": "custom-service",
  "provider": "openai",
  "api_key": "your-api-key",
  "model": "custom-model",
  "base_url": "https://your-custom-api.com/v1",
  "is_default": true,
  "is_active": true
}
```

## 优势

1. **统一管理** - 所有 LLM 配置通过前端 LLM 管理页面统一管理
2. **热加载** - 配置更新后无需重启服务,立即生效
3. **广泛支持** - 支持 20+ 种主流 LLM 提供商
4. **自动适配** - 自动配置各提供商的 API 端点,简化配置
5. **兼容性强** - 支持所有 OpenAI API 兼容的服务
6. **本地支持** - 内置 Ollama 支持,可使用本地模型

## 技术细节

### BaseURL 自动配置规则

1. 用户自定义 BaseURL > 提供商默认 BaseURL > OpenAI 默认值
2. Anthropic 通过原生 SDK 的 `WithBaseURL()` 选项配置
3. 其他提供商通过 OpenAI SDK 的 `WithBaseURL()` 选项配置

### 错误处理

- 配置验证失败时返回友好错误信息
- LLM 创建失败不影响服务启动,只记录警告
- 热加载失败记录日志但不中断请求处理

### 性能优化

- LLM 客户端按会话缓存,避免重复创建
- 配置更新时仅重新加载受影响的 Agent 实例
- 数据库查询结果在内存中缓存

## 下一步优化方向

1. **配置验证增强** - 添加 API Key 和模型的实时验证
2. **推荐模型 API** - 前端根据提供商动态显示推荐模型
3. **使用统计** - 记录各 LLM 的调用次数和 token 消耗
4. **负载均衡** - 支持多个 LLM 配置的自动切换和负载均衡
5. **成本控制** - 基于 token 消耗的预算管理和告警
