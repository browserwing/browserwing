# 工具调用说明提示词改进

## 背景

在 AI Agent 调用工具时，会通过 `instructions` 参数向用户解释：
- 为什么调用这个工具
- 期望达成什么目标

这个说明会展示给用户，帮助他们理解 AI 的思考过程。

## 问题

### 旧的提示词

```
"Please briefly explain: 1) Why you are calling this tool 2) What information or task you expect to accomplish with this tool. This explanation will be shown to users to help them understand the AI's thinking process. 3) In the explanation, use the specific tool name instead of saying 'this tool'. Respond in the same language as the user's message."
```

### 存在的问题

1. **太机械化**
   - 像在填表格：1) ... 2) ... 3) ...
   - 缺少人情味

2. **没有第一人称**
   - 不使用 "我"，显得冷冰冰
   - 像在写技术文档，不像在对话

3. **格式要求不清晰**
   - 没有明确示例
   - 可能产生过长或过短的说明

4. **用户体验不佳**
   ```
   ❌ "This tool will navigate to the URL and retrieve information."
   ❌ "Using browser_click tool to click element."
   ```
   这些说明太机械，用户看了没有温度感。

## 解决方案

### 新的提示词

```go
const instructionsDescription = `Write a brief, friendly explanation (1-2 sentences) in first person that tells the user what you're about to do and why. 

Guidelines:
- Use "I" or "I'm" to make it personal and natural
- Mention the specific tool name you're using (e.g., "I'm using browser_navigate to...")
- Focus on what you hope to accomplish for the user
- Keep it conversational and warm, like talking to a friend
- Match the user's language

Good examples:
- "I'm using browser_navigate to open Baidu so I can help you search for the latest AI news."
- "Let me use browser_click to click that login button for you."
- "I'm going to use web_search to find today's trending GitHub repositories."

Bad examples:
- "This tool will navigate to the URL." (too mechanical, no "I")
- "Using tool to accomplish task." (too robotic)
- "I will call this tool." (don't say "this tool", name it specifically)`
```

## 改进点

### 1. 使用第一人称 ✅

**之前**:
```
"This tool will navigate to Baidu."
"Using browser_click to click button."
```

**现在**:
```
"I'm using browser_navigate to open Baidu..."
"Let me use browser_click to click that button for you."
```

### 2. 更友好和对话式 ✅

**之前**:
```
"Navigate to URL to retrieve information."
```

**现在**:
```
"I'm opening Baidu so I can help you search for the latest AI news."
```

### 3. 明确的格式要求 ✅

- **长度**: 1-2 sentences（简洁但完整）
- **风格**: conversational and warm, like talking to a friend
- **必须包含**: 工具名称 + 目的

### 4. 提供具体示例 ✅

**Good examples** (正面示例):
- ✅ "I'm using browser_navigate to open Baidu so I can help you search for the latest AI news."
- ✅ "Let me use browser_click to click that login button for you."
- ✅ "I'm going to use web_search to find today's trending GitHub repositories."

**Bad examples** (反面示例):
- ❌ "This tool will navigate to the URL." (太机械，没有 "I")
- ❌ "Using tool to accomplish task." (太机器人)
- ❌ "I will call this tool." (不要说 "this tool"，要具体说工具名)

## 预期效果

### 示例场景 1: 浏览器导航

**用户请求**: "打开百度搜索人工智能新闻"

**旧的 instructions**:
```
"This tool will navigate to Baidu.com to perform the search operation."
```
😐 感觉像在读机器说明书

**新的 instructions**:
```
"我正在使用 browser_navigate 打开百度，这样就能帮你搜索最新的人工智能新闻了。"
```
😊 像朋友在帮忙，有温度

### 示例场景 2: 点击操作

**用户请求**: "点击登录按钮"

**旧的 instructions**:
```
"Using browser_click tool to click the login button element."
```
😐 技术性太强

**新的 instructions**:
```
"让我用 browser_click 帮你点击这个登录按钮。"
```
😊 简单友好，像在对话

### 示例场景 3: 搜索信息

**用户请求**: "今天 GitHub 上有什么热门项目？"

**旧的 instructions**:
```
"Tool will search for trending GitHub repositories."
```
😐 冷冰冰

**新的 instructions**:
```
"I'm using web_search to find today's trending GitHub repositories for you."
```
😊 主动且友好

## 语言匹配

提示词中强调：**Match the user's language**

### 中文用户
```
"我正在使用 browser_navigate 打开百度..."
"让我用 browser_click 帮你点击..."
```

### 英文用户
```
"I'm using browser_navigate to open Baidu..."
"Let me use browser_click to click that button for you."
```

### 日文用户
```
"browser_navigate を使用して百度を開きます..."
"browser_click でそのボタンをクリックしますね。"
```

## 对比总结

| 方面 | 旧提示词 | 新提示词 |
|------|---------|---------|
| **人称** | 第三人称/被动语态 | 第一人称 "I", "I'm" ✅ |
| **语气** | 机械、技术性 | 友好、对话式 ✅ |
| **长度要求** | 不明确 | 1-2 sentences ✅ |
| **示例** | 无 | 有正反面示例 ✅ |
| **格式** | 列表式 1) 2) 3) | 自然对话 ✅ |
| **温度感** | 冷 ❄️ | 温暖 🌟 ✅ |

## 实际效果预测

### 用户体验提升

**场景**: 用户让 AI 帮忙填写表单

**旧体验**:
```
用户: "帮我填写这个注册表单"
AI: [调用 browser_type]
说明: "Using browser_type tool to input text into form field."
用户: 😐 (感觉像在用冷冰冰的脚本)
```

**新体验**:
```
用户: "帮我填写这个注册表单"
AI: [调用 browser_type]
说明: "我正在使用 browser_type 帮你填写姓名字段。"
用户: 😊 (感觉像有个助手在帮忙)
```

### 信任度提升

使用第一人称和友好语气，会让用户感觉：
- ✅ AI 更像一个助手，而不是工具
- ✅ 操作更透明，知道 AI 在做什么
- ✅ 更容易建立信任关系

## 技术实现

### 修改位置

**文件**: `/root/code/browserpilot/backend/agent/tools/init.go`

**函数**: `WrapTool()` 使用这个常量作为 instructions 参数的描述

### 工具包装器

```go
func WrapTool(tool interfaces.Tool) interfaces.Tool {
    schema := tool.InputSchema()
    if schema == nil {
        schema = make(map[string]interface{})
    }
    
    // 添加 instructions 参数
    if properties, ok := schema["properties"].(map[string]interface{}); ok {
        properties["instructions"] = map[string]interface{}{
            "type":        "string",
            "description": instructionsDescription, // 使用新的提示词
        }
        // ...
    }
    
    return &ToolWrapper{
        tool:     tool,
        schema:   schema,
        required: required,
    }
}
```

## 测试建议

### 1. 手动测试

测试不同的工具调用，检查 instructions 是否友好：

```bash
# 浏览器导航
curl -X POST http://localhost:8080/api/agent/chat \
  -d '{"message": "打开百度搜索AI新闻"}'

# 期望看到类似：
# "我正在使用 browser_navigate 打开百度，这样就能帮你搜索AI新闻了。"
```

### 2. 多语言测试

测试不同语言的 instructions：

```bash
# 中文
"打开百度" → "我正在使用 browser_navigate..."

# 英文
"Open Baidu" → "I'm using browser_navigate..."

# 日文
"百度を開いて" → "browser_navigate を使用して..."
```

### 3. 检查要点

- [ ] 使用第一人称 ("I", "I'm", "Let me", "我")
- [ ] 提到具体工具名称 (不说 "this tool")
- [ ] 长度合理 (1-2 sentences)
- [ ] 语气友好 (像和朋友聊天)
- [ ] 匹配用户语言

## 后续优化建议

### 1. 添加个性化

可以让不同的 AI 助手有不同的说话风格：

```go
// 专业助手
"I'm using browser_navigate to access Baidu for your search request."

// 友好助手
"Let me open Baidu for you using browser_navigate! 😊"

// 简洁助手
"Opening Baidu with browser_navigate to search."
```

### 2. 根据任务类型调整

```go
// 简单任务
"I'm using web_search to find that for you."

// 复杂任务
"I'm using browser_navigate to start this multi-step process. First, I'll open the website..."
```

### 3. 添加进度感

对于多步骤任务：

```go
// 第一步
"I'm using browser_navigate to open Baidu. (Step 1/3)"

// 第二步
"Now I'm using browser_type to enter your search term. (Step 2/3)"

// 第三步
"Finally, I'm using browser_click to submit the search. (Step 3/3)"
```

## 总结

### ✅ 主要改进

1. **第一人称视角** - 使用 "I", "I'm", "Let me"
2. **友好对话式** - 像朋友聊天，不是技术文档
3. **明确示例** - 提供正反面对比
4. **长度控制** - 1-2 sentences，简洁但完整
5. **情感温度** - 从冷冰冰到温暖友好

### 📊 预期收益

- **用户满意度** ⬆️ - 更有人情味的交互
- **信任度** ⬆️ - 透明且友好的说明
- **理解度** ⬆️ - 清楚知道 AI 在做什么
- **体验一致性** ⬆️ - 所有工具调用都有统一的友好风格

### 🎯 核心理念

> **让 AI 助手像人一样说话，而不是像机器人一样汇报。**

从 "Using tool X to perform action Y"
到 "I'm using tool X to help you with Y" 🎉

## 相关文件

- `/root/code/browserpilot/backend/agent/tools/init.go` - 主要修改
- `/root/code/browserpilot/backend/agent/agent.go` - 使用 instructions
- `/root/code/browserpilot/docs/TOOL_INSTRUCTIONS_IMPROVEMENT.md` - 本文档
