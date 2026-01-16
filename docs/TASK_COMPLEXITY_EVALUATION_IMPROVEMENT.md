# 任务复杂度评估提示词改进

## 问题描述

### 原始日志
```json
{
  "level": "info",
  "msg": "[TaskEval] Raw response: ```json\n{\n  \"complex_mode\": \"simple\",\n  \"reasoning\": \"The task involves a straightforward sequence of actions: opening a browser, navigating to Baidu, searching for a specific term, and clicking a link.\",\n  \"confidence\": \"high\",\n  \"explanation\": \"这是一个简单的任务，只需要依次完成打开浏览器、搜索和点击链接的操作。\"\n}\n```",
  "time": "2026-01-16 22:02:10"
}
```

### 问题分析

**用户任务**: 打开浏览器 → 导航到百度 → 搜索特定词 → 点击链接

**预期工具调用**:
1. `browser_navigate` - 导航到百度
2. `browser_get_semantic_tree` - 获取页面元素
3. `browser_type` - 在搜索框输入
4. `browser_press_key` 或 `browser_click` - 提交搜索
5. `browser_click` - 点击搜索结果

**总计**: 4-5 个工具调用 → **应该是 MEDIUM 任务**

**实际评估**: SIMPLE ❌

### 根本原因

旧提示词的问题：

1. **分类标准模糊**
   - SIMPLE: "1-3 tool calls"
   - MEDIUM: "4-7 tool calls"
   - COMPLEX: "7+ tool calls"
   - 但对浏览器自动化任务缺乏明确指导

2. **示例不够具体**
   - MEDIUM 和 COMPLEX 的示例太相似
   - 缺少具体的浏览器操作示例

3. **没有强调工具调用计数**
   - LLM 可能基于"任务描述的复杂度"而不是"预期工具调用次数"来判断

## 解决方案

### 新的提示词设计

#### 1. 明确的分类标准

```
**SIMPLE (1-3 tool calls):**
- Single information queries (no browser automation)
- Direct API calls or searches

**MEDIUM (4-7 tool calls):**
- Browser automation with multiple steps
- Multi-step information gathering and processing

**COMPLEX (8+ tool calls):**
- Multi-page workflows with data processing
- Complex form filling with validation
- Comprehensive data extraction and analysis
```

#### 2. 具体的浏览器操作示例

**SIMPLE 示例**:
- "Search for today's trending GitHub repositories" → 1 call
- "What's the weather in Beijing?" → 1 call

**MEDIUM 示例**:
- "Open Baidu, search for 'AI news', click the first result" → 4-5 calls
- "Go to GitHub trending page and get the top 3 projects" → 4-5 calls
- "Fill a simple form and submit" → 4-6 calls

**COMPLEX 示例**:
- "Compare prices across 3 e-commerce sites and create a summary" → 12+ calls
- "Automate a complete user registration flow with email verification" → 10+ calls
- "Scrape data from multiple pages and generate a report" → 15+ calls

#### 3. 重要提示

```
**Important Notes:**
- Browser automation tasks (navigate, type, click, etc.) are usually MEDIUM or COMPLEX, rarely SIMPLE
- Count each browser operation separately: navigate=1, type=1, click=1, get_semantic_tree=1
- If the task mentions "open browser", "search", "click", it's likely 4+ tool calls (MEDIUM)
```

### 完整的新提示词

```go
evalPrompt := fmt.Sprintf(`Analyze the following user request and estimate the number of tool calls required, then classify the task complexity.

User request: "%s"

Classification Guidelines (based on estimated tool calls):

**SIMPLE (1-3 tool calls):**
- Single information queries (no browser automation)
- Direct API calls or searches
- Examples:
  * "Search for today's trending GitHub repositories" → 1 call (web_search)
  * "What's the weather in Beijing?" → 1 call (web_search)
  * "Calculate the result of 123 * 456" → 1 call (calculate)

**MEDIUM (4-7 tool calls):**
- Browser automation with multiple steps
- Multi-step information gathering and processing
- Examples:
  * "Open Baidu, search for 'AI news', click the first result" → 4-5 calls (navigate, type, press_key, click)
  * "Go to GitHub trending page and get the top 3 projects" → 4-5 calls (navigate, get_semantic_tree, extract)
  * "Fill a simple form and submit" → 4-6 calls (navigate, type×3, click)

**COMPLEX (8+ tool calls):**
- Multi-page workflows with data processing
- Complex form filling with validation
- Comprehensive data extraction and analysis
- Examples:
  * "Compare prices across 3 e-commerce sites and create a summary" → 12+ calls
  * "Automate a complete user registration flow with email verification" → 10+ calls
  * "Scrape data from multiple pages and generate a report" → 15+ calls

**Important Notes:**
- Browser automation tasks (navigate, type, click, etc.) are usually MEDIUM or COMPLEX, rarely SIMPLE
- Count each browser operation separately: navigate=1, type=1, click=1, get_semantic_tree=1
- If the task mentions "open browser", "search", "click", it's likely 4+ tool calls (MEDIUM)

Response format (JSON only, no explanation, no markdown):
{
  "complex_mode": "simple/medium/complex",
  "reasoning": "Brief explanation with estimated tool call count",
  "confidence": "high/medium/low",
  "explanation": "Short user-friendly explanation in Chinese"
}`, userMessage)
```

## 改进效果

### 预期结果

对于任务 "打开浏览器、导航到百度、搜索、点击链接"：

**旧提示词**:
```json
{
  "complex_mode": "simple",
  "reasoning": "The task involves a straightforward sequence of actions",
  "confidence": "high"
}
```

**新提示词 (预期)**:
```json
{
  "complex_mode": "medium",
  "reasoning": "Requires 4-5 tool calls: navigate, type, press_key/click, click. Browser automation with multiple steps.",
  "confidence": "high"
}
```

### Agent 配置对应关系

| 复杂度 | Max Iterations | 适用场景 |
|--------|---------------|----------|
| SIMPLE | 3 | 单次查询、简单计算 |
| MEDIUM | 7 | 浏览器自动化、多步骤流程 |
| COMPLEX | 12 | 多页面工作流、数据处理 |

## 其他改进

### 1. 添加 Medium Agent

```go
const (
    maxIterationsSimple  = 3
    maxIterationsMedium  = 7  // 新增
    maxIterationsComplex = 12
)

type AgentInstances struct {
    SimpleAgent  *agent.Agent
    MediumAgent  *agent.Agent  // 新增
    ComplexAgent *agent.Agent
    EvalAgent    *agent.Agent
}
```

### 2. Agent 选择逻辑

```go
switch complexity.ComplexMode {
case ComplexModeComplex:
    ag = agentInstances.ComplexAgent
    logger.Info(ctx, "Using COMPLEX agent (max iterations: %d)", maxIterationsComplex)
case ComplexModeMedium:
    ag = agentInstances.MediumAgent
    logger.Info(ctx, "Using MEDIUM agent (max iterations: %d)", maxIterationsMedium)
default:
    ag = agentInstances.SimpleAgent
    logger.Info(ctx, "Using SIMPLE agent (max iterations: %d)", maxIterationsSimple)
}
```

## 测试建议

### 测试用例

#### 应该评估为 SIMPLE 的任务
1. "今天的 GitHub trending 有什么项目？" → web_search
2. "北京今天天气怎么样？" → web_search
3. "计算 123 * 456" → calculate

#### 应该评估为 MEDIUM 的任务
1. "打开百度，搜索'人工智能新闻'，点击第一个结果" → 4-5 calls
2. "去 GitHub trending 页面，获取前 3 个项目" → 4-5 calls
3. "填写一个注册表单并提交" → 4-6 calls
4. "在某个网站搜索产品，获取前 5 个结果" → 5-6 calls

#### 应该评估为 COMPLEX 的任务
1. "在 3 个电商网站比较同一个产品的价格，生成总结报告" → 12+ calls
2. "自动化完成用户注册流程，包括邮箱验证" → 10+ calls
3. "从多个页面抓取数据，处理后生成表格" → 15+ calls
4. "监控网站变化，定期截图并对比" → 8+ calls

### 验证方法

1. **查看日志**:
   ```bash
   tail -f logs/app.log | grep "TaskEval"
   ```

2. **检查 Agent 选择**:
   ```bash
   tail -f logs/app.log | grep "Using.*agent"
   ```

3. **统计工具调用次数**:
   ```bash
   tail -f logs/app.log | grep "ToolCall Event"
   ```

## 关键改进点总结

### ✅ 改进前
- ❌ 浏览器自动化任务被错误评估为 SIMPLE
- ❌ 只有 SIMPLE 和 COMPLEX 两档，缺少 MEDIUM
- ❌ 提示词缺少具体的工具调用计数指导
- ❌ 示例不够明确

### ✅ 改进后
- ✅ 明确指出浏览器自动化通常是 MEDIUM 或 COMPLEX
- ✅ 新增 MEDIUM 档位（7次迭代）
- ✅ 强调按预期工具调用次数分类
- ✅ 提供具体的浏览器操作示例
- ✅ 每个示例都标注预期调用次数
- ✅ 添加"Important Notes"强调关键判断标准

## 预期收益

1. **更准确的任务评估**
   - 浏览器自动化任务不会再被错误归类为 SIMPLE
   - 评估更加量化和可预测

2. **更合理的迭代次数**
   - SIMPLE 任务: 3 次（足够）
   - MEDIUM 任务: 7 次（适中，避免过早 final call）
   - COMPLEX 任务: 12 次（充足）

3. **更好的用户体验**
   - 避免 SIMPLE agent 因迭代不足导致任务失败
   - 避免 COMPLEX agent 浪费不必要的迭代

## 相关文件

- `/root/code/browserpilot/backend/agent/agent.go` - 主要修改
- `/root/code/browserpilot/backend/executor/mcp_tools.go` - 修复返回格式

## 后续优化建议

1. **收集真实数据**
   - 记录每个任务的实际工具调用次数
   - 根据数据调整分类阈值

2. **动态调整**
   - 如果 MEDIUM agent 经常超时，考虑增加到 8-9 次
   - 如果 SIMPLE agent 经常提前完成，考虑减少到 2 次

3. **添加任务类型标签**
   - 除了复杂度，还可以标记任务类型：browser, search, calculation, data_processing
   - 不同类型使用不同的 agent 配置

4. **学习历史表现**
   - 记录相似任务的历史评估和实际表现
   - 使用历史数据改进评估准确性
