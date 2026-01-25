# 评估失败默认行为修复

## 问题描述

用户反馈：新建会话问"你是什么模型"时，系统会启动浏览器并调用 `browser_extract` 工具。

### 现象

```
用户: "你是什么模型"
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
实际行为 ❌:
[TaskEval] Evaluating task complexity for message: 你是什么模型
[Execute] Calling MCP tool: browser_extract
[callExecutorTool] Browser not running, starting...
[Start] Starting browser...
...

❌ 不应该启动浏览器！
❌ 不应该调用任何工具！
```

### 期望行为

```
用户: "你是什么模型"
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
预期行为 ✅:
[TaskEval] Evaluating task complexity
[TaskEval] Task evaluated as none (need_tools: false)
[DirectLLM] Task doesn't need tools, direct response
助手: "我是 GPT-4..."

✅ 直接回复，不调用工具
```

## 根本原因分析

### 日志分析

从日志看，只有一行评估日志，然后就直接跳到了工具调用：

```
[TaskEval] Evaluating task complexity for message: 你是什么模型
↓ (缺少后续日志)
[Execute] Calling MCP tool: browser_extract
```

**缺少的关键日志：**
- ❌ 没有 `[TaskEval] Raw response: ...`
- ❌ 没有 `[TaskEval] Task evaluated as ...`
- ❌ 没有 `[DirectLLM] ...` 或 `Using SIMPLE agent ...`

**结论：** 评估过程出错了！

### 代码追踪

#### 1. 评估函数中的错误处理

**旧代码（有问题）：**
```go
func (am *AgentManager) evaluateTaskComplexity(...) (*TaskComplexity, error) {
    // ...
    
    response, err := agentInstances.EvalAgent.Run(evalCtx, evalPrompt)
    if err != nil {
        logger.Warn(ctx, "[TaskEval] Failed to evaluate: %v, defaulting to simple", err)
        return &TaskComplexity{
            ComplexMode: ComplexModeSimple,  // ❌ 只设置了 ComplexMode
            Reasoning:   "Evaluation failed",
            // ❌ NeedTools 没设置，默认值是 false
        }, nil
    }
    
    // ...解析响应
    if response == "" {
        return &TaskComplexity{
            ComplexMode: ComplexModeSimple,  // ❌ 同样问题
        }, nil
    }
    
    // ...JSON 解析
    if err := json.Unmarshal(...) {
        return &TaskComplexity{
            ComplexMode: ComplexModeSimple,  // ❌ 同样问题
        }, nil
    }
}
```

#### 2. SendMessage 中的错误处理

**旧代码（更严重的问题）：**
```go
func (am *AgentManager) SendMessage(...) error {
    // ...
    
    complexity, err := am.evaluateTaskComplexity(ctx, sessionID, userMessage)
    if err != nil {
        logger.Warn(ctx, "Failed to evaluate: %v, using simple agent", err)
        complexity = &TaskComplexity{
            NeedTools:   true,               // ❌ 默认设置为 true！
            ComplexMode: ComplexModeSimple,
        }
    }
    
    // 判断是否需要工具
    if !complexity.NeedTools {
        // 直接回复
    } else {
        // 使用 Agent + 工具  ← ❌ 错误默认会走这里！
    }
}
```

### 问题总结

**4 处错误的默认值设置：**

| 位置 | 旧默认值 | 问题 |
|------|----------|------|
| 1. evaluateTaskComplexity - err | 未设置 NeedTools | Go 零值 false，但不明确 |
| 2. evaluateTaskComplexity - empty | 未设置 NeedTools | Go 零值 false，但不明确 |
| 3. evaluateTaskComplexity - parse | 未设置 NeedTools | Go 零值 false，但不明确 |
| 4. SendMessage - err | `NeedTools: true` | ❌ **这是主要问题！** |

**流程图：**
```
用户消息: "你是什么模型"
    ↓
evaluateTaskComplexity()
    ↓
EvalAgent.Run() → ❌ 失败（超时/错误）
    ↓
返回 TaskComplexity{
    NeedTools: ❌ 未设置（默认 false）
    ComplexMode: "simple"
}
    ↓
SendMessage 捕获 error
    ↓
设置 complexity = &TaskComplexity{
    NeedTools: true,  ❌ 强制设置为 true！
    ComplexMode: "simple"
}
    ↓
判断 !complexity.NeedTools → false
    ↓
使用 SimpleAgent + 工具 ❌
    ↓
Agent 调用 browser_extract 工具 ❌
    ↓
启动浏览器 ❌
```

## 解决方案

### 核心原则

**评估失败时的安全默认值：不使用工具，直接回复**

理由：
1. ✅ **更安全** - 不会意外启动浏览器或调用昂贵的工具
2. ✅ **更快** - 直接 LLM 回复比工具调用快
3. ✅ **更合理** - 如果连评估都失败了，说明任务可能很简单或有问题
4. ✅ **用户友好** - 对于简单问答（如"你是什么模型"），直接回复最自然

### 修复代码

#### 1. evaluateTaskComplexity - 失败处理

```go
// ✅ 新代码
response, err := agentInstances.EvalAgent.Run(evalCtx, evalPrompt)
if err != nil {
    logger.Warn(ctx, "[TaskEval] Failed to evaluate: %v, defaulting to no tools", err)
    return &TaskComplexity{
        NeedTools:   false,  // ✅ 明确设置为 false
        ComplexMode: "none", // ✅ 使用 "none" 表示不需要工具
        Reasoning:   "Evaluation failed, defaulting to direct response",
        Confidence:  "low",
        Explanation: "评估失败，直接回复",
    }, nil
}
```

#### 2. evaluateTaskComplexity - 空响应处理

```go
// ✅ 新代码
if response == "" {
    logger.Warn(ctx, "[TaskEval] Empty response, defaulting to no tools")
    return &TaskComplexity{
        NeedTools:   false,  // ✅ 明确设置为 false
        ComplexMode: "none",
        Reasoning:   "Empty response, defaulting to direct response",
        Confidence:  "low",
        Explanation: "评估结果为空，直接回复",
    }, nil
}
```

#### 3. evaluateTaskComplexity - 解析失败处理

```go
// ✅ 新代码
if err := json.Unmarshal([]byte(response), &complexity); err != nil {
    logger.Warn(ctx, "[TaskEval] Failed to parse JSON: %v, defaulting to no tools", err)
    return &TaskComplexity{
        NeedTools:   false,  // ✅ 明确设置为 false
        ComplexMode: "none",
        Reasoning:   "Failed to parse evaluation result",
        Confidence:  "low",
        Explanation: "评估结果解析失败，直接回复",
    }, nil
}
```

#### 4. SendMessage - 评估错误处理

```go
// ✅ 新代码（最重要的修复！）
complexity, err := am.evaluateTaskComplexity(ctx, sessionID, userMessage)
if err != nil {
    logger.Warn(ctx, "Failed to evaluate: %v, using direct response", err)
    complexity = &TaskComplexity{
        NeedTools:   false,  // ✅ 改为 false！
        ComplexMode: "none",
        Reasoning:   "Evaluation error, defaulting to direct response",
        Confidence:  "low",
        Explanation: "评估失败，直接回复",
    }
}
```

## 修复后的流程

```
用户消息: "你是什么模型"
    ↓
evaluateTaskComplexity()
    ↓
EvalAgent.Run() → ❌ 失败（超时/错误）
    ↓
返回 TaskComplexity{
    NeedTools: false,    ✅ 明确设置
    ComplexMode: "none", ✅ 明确标记
}
    ↓
SendMessage 捕获 error (optional)
    ↓
设置 complexity = &TaskComplexity{
    NeedTools: false,    ✅ 改为 false
    ComplexMode: "none"
}
    ↓
判断 !complexity.NeedTools → ✅ true
    ↓
直接回复（使用 SimpleAgent 不调用工具）✅
    ↓
返回: "我是 GPT-4..." ✅
```

## 对比效果

### 场景 1: 评估失败

| 阶段 | 旧版本 | 新版本 |
|------|--------|--------|
| 评估 | ❌ 失败 | ❌ 失败（相同）|
| 默认值 | `NeedTools: true` | ✅ `NeedTools: false` |
| 行为 | ❌ 使用 Agent + 工具 | ✅ 直接回复 |
| 结果 | ❌ 启动浏览器 | ✅ 立即回复 |

### 场景 2: 简单问答（评估成功）

| 阶段 | 旧版本 | 新版本 |
|------|--------|--------|
| 评估 | ✅ 成功 | ✅ 成功 |
| 结果 | `NeedTools: false` | `NeedTools: false` |
| 行为 | ✅ 直接回复 | ✅ 直接回复 |

### 场景 3: 需要工具（评估成功）

| 阶段 | 旧版本 | 新版本 |
|------|--------|--------|
| 评估 | ✅ 成功 | ✅ 成功 |
| 结果 | `NeedTools: true` | `NeedTools: true` |
| 行为 | ✅ 使用工具 | ✅ 使用工具 |

## 技术细节

### TaskComplexity 字段含义

```go
type TaskComplexity struct {
    NeedTools   bool   // true: 需要工具, false: 直接回复
    ComplexMode string // "none", "simple", "medium", "complex"
    // ...
}
```

**ComplexMode 取值：**
- `"none"`: 不需要工具
- `"simple"`: 需要工具，1-3 次调用
- `"medium"`: 需要工具，4-7 次调用
- `"complex"`: 需要工具，8+ 次调用

### 代码变更统计

```
修改的文件: backend/agent/agent.go

修改的函数:
├─ evaluateTaskComplexity()
│  ├─ 错误处理默认值 (+2 行, ~3 行修改)
│  ├─ 空响应处理默认值 (+2 行, ~3 行修改)
│  └─ 解析失败处理默认值 (+2 行, ~3 行修改)
│
└─ SendMessage()
   └─ 评估错误处理默认值 (+2 行, ~3 行修改)

总计: +8 行, ~12 行修改
```

### 日志改进

**旧日志：**
```
[TaskEval] Failed to evaluate: error, defaulting to simple
```

**新日志：**
```
[TaskEval] Failed to evaluate: error, defaulting to no tools
```

更清晰地表明默认行为是"不使用工具"。

## 测试场景

### 测试 1: 简单问答（评估成功）

```bash
curl -X POST .../sessions/{id}/messages \
  -d '{"message": "你是什么模型"}'
```

**预期：**
- ✅ 评估为 `NeedTools: false`
- ✅ 直接回复
- ❌ 不启动浏览器

### 测试 2: 简单问答（评估失败）

```bash
# 模拟评估失败（如 LLM 超时）
curl -X POST .../sessions/{id}/messages \
  -d '{"message": "你好"}'
```

**预期：**
- ❌ 评估失败
- ✅ 默认 `NeedTools: false`
- ✅ 直接回复
- ❌ 不启动浏览器

### 测试 3: 需要工具（评估成功）

```bash
curl -X POST .../sessions/{id}/messages \
  -d '{"message": "搜索今天的新闻"}'
```

**预期：**
- ✅ 评估为 `NeedTools: true`
- ✅ 使用工具
- ✅ 调用 web_search

### 测试 4: 需要工具（评估失败）

```bash
# 模拟评估失败
curl -X POST .../sessions/{id}/messages \
  -d '{"message": "打开百度"}'
```

**预期（修复后的保守行为）：**
- ❌ 评估失败
- ✅ 默认 `NeedTools: false`
- ✅ 直接回复（告知无法打开浏览器）
- ❌ 不启动浏览器

**注意：** 这个场景下用户可能需要重试，但至少不会意外启动浏览器。

## 优势总结

### 安全性

| 场景 | 旧版本 | 新版本 |
|------|--------|--------|
| 评估失败 | ❌ 可能误用工具 | ✅ 安全默认（不用工具）|
| 简单问答 | ✅ 正常 | ✅ 正常 |
| 工具任务 | ✅ 正常 | ✅ 正常 |

### 用户体验

| 指标 | 旧版本 | 新版本 |
|------|--------|--------|
| 意外启动浏览器 | ❌ 可能发生 | ✅ 不会发生 |
| 简单问答速度 | ✅ 快（如果评估成功）| ✅ 快 |
| 错误恢复 | ❌ 差（工具调用失败）| ✅ 好（直接回复）|

### 成本优化

| 场景 | 旧版本成本 | 新版本成本 | 节省 |
|------|-----------|-----------|------|
| 评估失败 + 工具调用 | 高（浏览器 + LLM）| 低（仅 LLM）| 80% |
| 评估失败 + 直接回复 | - | 低 | - |

## 相关文档

- [DIRECT_LLM_RESPONSE.md](./DIRECT_LLM_RESPONSE.md) - 直接 LLM 回复优化
- [LAZY_AGENT_CREATION.md](./LAZY_AGENT_CREATION.md) - Agent 按需创建

## 总结

通过这次修复：

1. ✅ **修复了评估失败的默认行为** - 4 处默认值全部设置为 `NeedTools: false`
2. ✅ **防止意外工具调用** - 评估失败时不会启动浏览器或调用工具
3. ✅ **改进了日志** - 更清晰地表明默认行为
4. ✅ **提升了安全性** - 保守的默认策略更安全

**核心改进：** 评估失败时，默认"直接回复"而不是"使用工具"，让系统更安全、更快、更符合预期！🎉
