# Ollama 支持增强总结

## 概述

完善了对 Ollama 本地 LLM 的支持，添加了工具调用模型识别和 API Key 可选配置。

## 背景

Ollama 是一个流行的本地 LLM 运行框架，支持在本地运行各种开源模型（如 Qwen、Llama、DeepSeek 等）。之前的代码已经支持 Ollama，但存在以下不完善之处：

### 原有支持情况

✅ **已支持的功能：**
- 默认 BaseURL 配置：`http://localhost:11434/v1`
- 推荐模型列表：qwen2.5、llama3.3、deepseek-r1、mistral
- 使用 OpenAI 兼容模式处理

❌ **存在的问题：**
1. `SupportsToolCalling` 函数中没有明确列出 Ollama 支持工具调用的模型
2. `ValidateLLMConfig` 要求所有提供商必须提供 API Key（但 Ollama 本地运行不需要）
3. 创建客户端时如果 API Key 为空可能导致问题

## 完成的改进

### 1. ✅ 添加 Ollama 工具调用模型列表

**文件：** `backend/agent/agent_llm.go` - `SupportsToolCalling` 函数

**改动：** 在 `supportedModels` map 中添加了 Ollama 支持工具调用的模型列表：

```go
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
```

**说明：**
- 包含了主流的支持工具调用的开源模型
- 支持模型名称的多种格式（如 `llama3.3` 和 `llama-3.3`）
- 涵盖了 Qwen、Llama、Mistral、DeepSeek 等主流系列

### 2. ✅ API Key 可选配置

**文件：** `backend/agent/agent_llm.go` - `ValidateLLMConfig` 函数

**改动前：**
```go
if config.APIKey == "" {
    return fmt.Errorf("api_key cannot be empty")
}
```

**改动后：**
```go
// Ollama 本地运行时不需要 API Key
provider := strings.ToLower(config.Provider)
if provider != "ollama" && config.APIKey == "" {
    return fmt.Errorf("api_key cannot be empty")
}
```

**说明：**
- Ollama 本地运行不需要 API Key 验证
- 其他提供商仍然要求必须提供 API Key
- 提高了配置的灵活性

### 3. ✅ 默认 API Key 占位符

**文件：** `backend/agent/agent_llm.go` - `createOpenAICompatibleClient` 函数

**改动：**
```go
// Ollama 本地运行时不需要真实的 API Key，提供默认值
apiKey := config.APIKey
if provider == "ollama" && apiKey == "" {
    apiKey = "ollama" // Ollama 本地不验证 API Key，提供占位符即可
}

client := openai.NewClient(apiKey, opts...)
```

**说明：**
- 当 Ollama 没有提供 API Key 时，使用 `"ollama"` 作为占位符
- 避免 OpenAI SDK 因为空 API Key 而报错
- Ollama 服务器不会验证这个值，只是满足 SDK 的非空要求

## 使用示例

### 配置 Ollama（不需要 API Key）

```json
{
  "provider": "ollama",
  "model": "qwen2.5:latest",
  "base_url": "http://localhost:11434/v1"
}
```

或者使用默认 BaseURL：

```json
{
  "provider": "ollama",
  "model": "llama3.3:latest"
}
```

### 配置 Ollama（自定义端口）

```json
{
  "provider": "ollama",
  "model": "deepseek-r1:latest",
  "base_url": "http://localhost:11435/v1"
}
```

### 工具调用支持检测

```go
// 检查模型是否支持工具调用
supported := SupportsToolCalling("ollama", "qwen2.5:latest")
// 返回 true - Qwen2.5 支持工具调用

supported = SupportsToolCalling("ollama", "llama3.3:latest")
// 返回 true - Llama 3.3 支持工具调用

supported = SupportsToolCalling("ollama", "deepseek-r1:latest")
// 返回 true - DeepSeek R1 支持工具调用
```

## Ollama 支持的工具调用模型

| 模型系列 | 示例模型名 | 工具调用支持 |
|---------|----------|-------------|
| **Qwen** | qwen2.5:latest, qwen2:latest | ✅ 支持 |
| **Llama 3** | llama3.3:latest, llama3.2:latest, llama3.1:latest | ✅ 支持 |
| **Mistral** | mistral:latest, mixtral:latest | ✅ 支持 |
| **DeepSeek** | deepseek-r1:latest, deepseek-v3:latest | ✅ 支持 |
| **Yi** | yi-coder:latest | ✅ 支持 |
| **Phi** | phi3:latest, phi4:latest | ✅ 支持 |
| **Gemma** | gemma2:latest | ✅ 支持 |
| **Command R** | command-r:latest, command-r-plus:latest | ✅ 支持 |

**注意：** 较旧的模型（如 Llama 2）可能不支持工具调用。

## Ollama 配置最佳实践

### 1. 基本配置（推荐）

```json
{
  "provider": "ollama",
  "model": "qwen2.5:latest",
  "api_key": ""  // 可以省略或留空
}
```

### 2. 指定端口

```json
{
  "provider": "ollama",
  "model": "llama3.3:latest",
  "base_url": "http://localhost:11434/v1"
}
```

### 3. 远程 Ollama 服务器

```json
{
  "provider": "ollama",
  "model": "deepseek-r1:latest",
  "base_url": "http://192.168.1.100:11434/v1"
}
```

### 4. 温度和其他参数

```json
{
  "provider": "ollama",
  "model": "qwen2.5:latest",
  "temperature": 0.7,
  "max_tokens": 4096
}
```

## 技术细节

### Ollama API 特性

1. **OpenAI 兼容**：Ollama 实现了 OpenAI API 的兼容接口
2. **本地运行**：默认在 `localhost:11434` 运行
3. **无需认证**：本地运行时不需要 API Key
4. **工具调用**：新版本模型支持 Function Calling

### BrowserWing 中的 Ollama 处理流程

```
用户配置 Ollama
    ↓
ValidateLLMConfig (API Key 可选)
    ↓
CreateLLMClient
    ↓
createOpenAICompatibleClient (提供默认 API Key)
    ↓
OpenAI SDK 客户端 (BaseURL: localhost:11434/v1)
    ↓
Ollama 服务器
```

### 与其他提供商的对比

| 特性 | Ollama | OpenAI | Anthropic | DeepSeek |
|------|--------|--------|-----------|----------|
| 本地运行 | ✅ 是 | ❌ 否 | ❌ 否 | ❌ 否 |
| 需要 API Key | ❌ 否 | ✅ 是 | ✅ 是 | ✅ 是 |
| OpenAI 兼容 | ✅ 是 | ✅ 是 | ❌ 否 | ✅ 是 |
| 工具调用 | ✅ 部分模型 | ✅ 是 | ✅ 是 | ✅ 是 |
| 特殊处理 | ✅ API Key可选 | ❌ 否 | ✅ 原生SDK | ✅ top_p限制 |

## 常见问题

### Q1: Ollama 是否需要 API Key？

**A:** 本地运行的 Ollama 不需要 API Key。如果你连接到远程 Ollama 服务器且配置了认证，可能需要提供 API Key。

### Q2: 如何知道我的模型是否支持工具调用？

**A:** 使用 `SupportsToolCalling("ollama", "模型名")` 函数检查。一般来说，较新的模型（如 Qwen2.5、Llama3.3、DeepSeek R1）都支持。

### Q3: 如果 Ollama 运行在非默认端口怎么办？

**A:** 在配置中指定 `base_url`：
```json
{
  "provider": "ollama",
  "model": "qwen2.5:latest",
  "base_url": "http://localhost:自定义端口/v1"
}
```

### Q4: Ollama 支持流式响应吗？

**A:** 是的，Ollama 通过 OpenAI 兼容接口支持流式响应（Server-Sent Events）。

### Q5: 如何在 BrowserWing 中测试 Ollama？

**A:** 
1. 确保 Ollama 已经运行：`ollama serve`
2. 拉取模型：`ollama pull qwen2.5`
3. 在 BrowserWing 配置中设置：
   ```json
   {
     "provider": "ollama",
     "model": "qwen2.5:latest"
   }
   ```
4. 测试连接即可

## 推荐的 Ollama 模型

### 1. Qwen2.5（推荐）⭐
- **模型：** `qwen2.5:latest` 或 `qwen2.5:7b`
- **优势：** 优秀的中文支持，工具调用稳定
- **适用：** 通用任务、浏览器自动化

### 2. Llama 3.3
- **模型：** `llama3.3:latest` 或 `llama3.3:70b`
- **优势：** 强大的推理能力，英文表现优秀
- **适用：** 复杂任务、多步骤推理

### 3. DeepSeek R1
- **模型：** `deepseek-r1:latest`
- **优势：** 强大的推理能力，长上下文支持
- **适用：** 复杂分析、代码生成

### 4. Mistral
- **模型：** `mistral:latest`
- **优势：** 高效、快速，资源占用低
- **适用：** 快速响应场景

## 性能建议

### 内存要求

| 模型大小 | 最小内存 | 推荐内存 |
|---------|---------|---------|
| 7B | 8 GB | 16 GB |
| 14B | 16 GB | 32 GB |
| 70B+ | 64 GB | 128 GB |

### GPU 加速

Ollama 支持 NVIDIA GPU 加速（CUDA）和 Apple Silicon（Metal）：

```bash
# 检查 GPU 使用情况
ollama ps

# 指定使用 GPU
CUDA_VISIBLE_DEVICES=0 ollama serve
```

### 并发处理

Ollama 支持并发请求，但受限于可用内存：

```bash
# 设置最大并发数
OLLAMA_MAX_LOADED_MODELS=2 ollama serve
```

## 文件改动总结

| 文件 | 改动类型 | 说明 |
|------|---------|------|
| backend/agent/agent_llm.go | ✅ 修改 | 添加 Ollama 工具调用支持 + API Key 可选 |
| docs/OLLAMA_SUPPORT_ENHANCEMENT.md | ✅ 新增 | 本文档 |

**代码统计：**
- ➕ 新增代码：~30 行
- 📝 新增 Ollama 支持模型：15+ 个
- ✅ 编译通过：成功

## 改进效果

### Before（之前）

```go
// 没有明确的 Ollama 工具调用支持
supportedModels := map[string][]string{
    "openai": {...},
    "anthropic": {...},
    // ❌ 没有 Ollama
}

// 所有提供商都需要 API Key
if config.APIKey == "" {
    return fmt.Errorf("api_key cannot be empty")  // ❌ Ollama 本地不需要
}

// 直接使用空 API Key
client := openai.NewClient(config.APIKey, opts...)  // ❌ 可能出错
```

### After（改进后）

```go
// ✅ 明确列出 Ollama 支持的模型
supportedModels := map[string][]string{
    "openai": {...},
    "anthropic": {...},
    "ollama": {
        "qwen2.5", "llama3.3", "deepseek-r1", ...
    },
}

// ✅ Ollama 不需要 API Key
if provider != "ollama" && config.APIKey == "" {
    return fmt.Errorf("api_key cannot be empty")
}

// ✅ 提供默认 API Key
apiKey := config.APIKey
if provider == "ollama" && apiKey == "" {
    apiKey = "ollama"
}
client := openai.NewClient(apiKey, opts...)
```

## 测试建议

### 1. 测试基本连接

```bash
# 启动 Ollama
ollama serve

# 拉取测试模型
ollama pull qwen2.5:latest

# 测试 BrowserWing 配置
curl -X POST 'http://localhost:8080/api/v1/agent/llm/set' \
  -H 'Content-Type: application/json' \
  -d '{
    "provider": "ollama",
    "model": "qwen2.5:latest"
  }'
```

### 2. 测试工具调用

创建 Agent 会话并测试浏览器自动化：

```bash
# 创建会话
curl -X POST 'http://localhost:8080/api/v1/agent/sessions' \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Ollama Test"
  }'

# 发送消息测试工具调用
curl -X POST 'http://localhost:8080/api/v1/agent/sessions/{session_id}/messages' \
  -H 'Content-Type: application/json' \
  -d '{
    "content": "请打开百度搜索并搜索BrowserWing"
  }'
```

### 3. 测试不同模型

```bash
# 测试 Llama3.3
ollama pull llama3.3:latest

# 测试 DeepSeek R1
ollama pull deepseek-r1:latest

# 分别配置测试
```

## 与 Agent 系统集成

Ollama 在 BrowserWing Agent 系统中的工作流程：

```
用户消息
    ↓
Agent 接收
    ↓
Ollama LLM 处理
    ↓
工具调用识别 (SupportsToolCalling)
    ↓
执行浏览器操作 (browser_navigate, browser_click, etc.)
    ↓
返回结果给 Ollama
    ↓
Ollama 生成响应
    ↓
流式返回给用户
```

## 优势总结

### 1. 隐私保护 🔒
- ✅ 数据不离开本地
- ✅ 无需担心 API 限制和审查
- ✅ 完全掌控模型行为

### 2. 成本优势 💰
- ✅ 无 API 调用费用
- ✅ 一次下载，无限使用
- ✅ 适合高频率自动化任务

### 3. 响应速度 ⚡
- ✅ 本地运行，低延迟
- ✅ 无网络波动影响
- ✅ GPU 加速支持

### 4. 灵活性 🔧
- ✅ 支持多种开源模型
- ✅ 可自定义模型参数
- ✅ 易于扩展和调试

## 总结

**完成度：** 100% ✅

成功完善了 Ollama 本地 LLM 的支持，主要改进包括：

1. ✅ **工具调用识别** - 明确列出 15+ 个支持工具调用的模型
2. ✅ **API Key 可选** - Ollama 本地运行不再强制要求 API Key
3. ✅ **默认占位符** - 自动提供默认 API Key，避免 SDK 错误

**关键特性：**
- 完全本地运行，保护隐私
- 无 API 费用，降低成本
- 支持主流开源模型的工具调用
- 配置简单，开箱即用

Ollama 现在已经是 BrowserWing 完全支持的 LLM 提供商！🎉
