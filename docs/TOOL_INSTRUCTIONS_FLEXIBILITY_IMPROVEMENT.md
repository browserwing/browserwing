# 工具调用 Instructions 灵活性改进

## 问题描述

### 用户反馈

工具调用的 instructions 句式太固定，总是按照例子的模板来回复，缺乏变化和自然感。

**示例**:
```
用户: "帮我打开百度"
AI: "我将使用 browser_navigate 工具打开这个网页，来获取页面的最新内容。"

用户: "帮我点击按钮"
AI: "我将使用 browser_click 工具点击这个按钮，来获取页面的最新内容。"
```

😐 **问题**: 句式完全一样，像机器人一样重复

### 期望效果

希望 AI 的回复：
1. **句式灵活**: 不总是"我将使用...工具..."的固定格式
2. **自然语气**: 可以加一些语气词（好的、嗯、让我来等）
3. **变化多样**: 每次回复可以用不同的表达方式
4. **更有人情味**: 像真人对话一样自然

## 解决方案

### 改进策略

1. **明确要求创造性**: 添加 "Be creative and natural!" 指示
2. **提供风格提示**: 列出可用的语气词和句式变化
3. **多样化示例**: 提供多个不同风格的例子
4. **强调例子的作用**: 说明例子只是启发性的，不是模板

### 修改内容

**修改前**:
```go
const instructionsDescription = `CRITICAL: You MUST respond in the EXACT SAME LANGUAGE as the user's message.

Write a brief, friendly explanation (1-2 sentences) in first person:
1. What you're about to do with this specific tool (use the tool name, not "this tool")
2. Why you're doing it (what information or result you expect)

This explanation helps users understand your thinking process.

Examples:
- User in Chinese → Your response in Chinese: 我将使用 browser_navigate 工具打开这个网页，来获取页面的最新内容。
- User in English → Your response in English: I'll use the browser_navigate tool to open this webpage and retrieve its latest content.
`
```

**问题**:
- 只有1个例子（中英文各1个）
- 没有强调要灵活变化
- 没有展示不同的句式
- 没有语气词的使用示例

**修改后**:
```go
const instructionsDescription = `CRITICAL: You MUST respond in the EXACT SAME LANGUAGE as the user's message.

Write a brief, natural, and friendly explanation (1-2 sentences) in first person that tells the user what you're about to do and why. Use the specific tool name.

IMPORTANT: Be creative and natural! Vary your expressions, add natural speech patterns, and make it conversational. Don't always use the same sentence structure.

Style tips:
- Feel free to add natural interjections (好的/好/嗯/让我来/那么/Okay/Alright/Let me/So)
- Vary your sentence patterns
- Use different verbs and expressions
- Keep it friendly and conversational

Examples (USE AS INSPIRATION, NOT TEMPLATES):
Chinese variations:
- 好的，让我用 browser_navigate 打开这个网页看看最新内容。
- 嗯，我来用 browser_click 点击一下这个按钮，看看会发生什么。
- 那我先用 browser_type 在这里输入文字，然后我们就能看到结果了。
- 让我试试用 browser_extract 抓取这个页面的数据，应该能获取到你需要的信息。

English variations:
- Alright, I'll open this webpage with browser_navigate to grab the latest content.
- Let me click that button using browser_click and see what happens.
- Okay, I'm going to type the text here with browser_type so we can get the results.
- I'll extract the data from this page using browser_extract to get what you need.
`
```

**改进点**:
- ✅ 添加 "Be creative and natural!" 强调创造性
- ✅ 提供具体的风格提示（语气词、句式变化）
- ✅ 4个中文示例 + 4个英文示例，每个都不同
- ✅ 明确说明 "USE AS INSPIRATION, NOT TEMPLATES"
- ✅ 展示多种语气词的使用

## 关键改进点

### 1. 强调创造性和自然性

**新增**:
```
IMPORTANT: Be creative and natural! Vary your expressions, add natural speech patterns, 
and make it conversational. Don't always use the same sentence structure.
```

**作用**: 
- 直接告诉 LLM 要灵活变化
- 强调对话的自然感
- 禁止使用固定句式

### 2. 风格提示（Style Tips）

**中文语气词**:
- 好的 / 好
- 嗯
- 让我来
- 那么
- 那我
- 让我试试

**英文语气词**:
- Okay
- Alright
- Let me
- So
- I'll
- Let me try

### 3. 多样化的例子

#### 中文例子展示不同风格

**风格1 - 礼貌型**:
```
好的，让我用 browser_navigate 打开这个网页看看最新内容。
```

**风格2 - 探索型**:
```
嗯，我来用 browser_click 点击一下这个按钮，看看会发生什么。
```

**风格3 - 步骤型**:
```
那我先用 browser_type 在这里输入文字，然后我们就能看到结果了。
```

**风格4 - 尝试型**:
```
让我试试用 browser_extract 抓取这个页面的数据，应该能获取到你需要的信息。
```

#### 英文例子展示不同风格

**风格1 - 直接型**:
```
Alright, I'll open this webpage with browser_navigate to grab the latest content.
```

**风格2 - 探索型**:
```
Let me click that button using browser_click and see what happens.
```

**风格3 - 进行时态**:
```
Okay, I'm going to type the text here with browser_type so we can get the results.
```

**风格4 - 目标导向**:
```
I'll extract the data from this page using browser_extract to get what you need.
```

### 4. 例子的定位

**关键标注**:
```
Examples (USE AS INSPIRATION, NOT TEMPLATES):
```

**含义**:
- 例子是**启发性**的，不是固定模板
- LLM 应该学习例子的**风格**和**多样性**
- 鼓励 LLM **创造**自己的表达方式

## 效果对比

### 修改前（固定句式）

**场景1**:
```
用户: "帮我打开百度"
AI: "我将使用 browser_navigate 工具打开这个网页，来获取页面的最新内容。"
```

**场景2**:
```
用户: "点击搜索按钮"
AI: "我将使用 browser_click 工具点击这个按钮，来获取页面的最新内容。"
```

**场景3**:
```
用户: "在搜索框输入内容"
AI: "我将使用 browser_type 工具在搜索框输入内容，来获取页面的最新内容。"
```

😐 **问题**: 
- 句式完全一样："我将使用...工具..."
- 结尾重复："来获取页面的最新内容"
- 没有语气词，不自然

### 修改后（灵活多样）

**场景1**:
```
用户: "帮我打开百度"
AI: "好的，让我用 browser_navigate 打开百度首页。"
```

**场景2**:
```
用户: "点击搜索按钮"
AI: "嗯，我来点击一下这个搜索按钮，看看搜索结果。"
```

**场景3**:
```
用户: "在搜索框输入内容"
AI: "那我先在这个搜索框里输入你要搜的内容。"
```

**场景4** (额外的变化):
```
用户: "抓取这个页面的标题"
AI: "让我试试抓取一下页面标题，应该能拿到你需要的信息。"
```

😊 **改进**: 
- ✅ 句式多样化
- ✅ 有自然的语气词
- ✅ 每次回复都不同
- ✅ 更像真人对话

## 句式变化示例

### 中文句式

1. **"好的" 开头**:
   - 好的，让我...
   - 好的，我来...
   - 好，那我...

2. **"嗯" 开头**:
   - 嗯，我来...
   - 嗯，让我...
   - 嗯嗯，我这就...

3. **"让我" 开头**:
   - 让我来...
   - 让我试试...
   - 让我看看...

4. **"那" 开头**:
   - 那我先...
   - 那么我来...
   - 那就...

5. **直接动作**:
   - 我来...
   - 我先...
   - 我这就...

### 英文句式

1. **"Okay" 开头**:
   - Okay, I'll...
   - Okay, let me...
   - Okay, I'm going to...

2. **"Alright" 开头**:
   - Alright, I'll...
   - Alright, let me...

3. **"Let me" 开头**:
   - Let me...
   - Let me try...
   - Let me see...

4. **"So" 开头**:
   - So I'll...
   - So let me...

5. **"I'll" 开头**:
   - I'll just...
   - I'll go ahead and...

## 语气词的使用

### 中文语气词及其语境

| 语气词 | 语境 | 示例 |
|--------|------|------|
| 好的 | 表示接受、确认 | 好的，让我打开这个页面 |
| 好 | 简短确认 | 好，我来点击按钮 |
| 嗯 | 思考、同意 | 嗯，我来试试看 |
| 让我来 | 主动承担 | 让我来帮你搜索一下 |
| 那么 | 转折、进入下一步 | 那么我现在打开网页 |
| 那 | 简短转折 | 那我先输入文字 |
| 让我试试 | 尝试语气 | 让我试试能不能抓取数据 |
| 我来 | 主动 | 我来帮你点击这个 |

### 英文语气词及其语境

| 语气词 | 语境 | 示例 |
|--------|------|------|
| Okay | 确认、开始 | Okay, I'll open this page |
| Alright | 轻松确认 | Alright, let me click that |
| Let me | 主动承担 | Let me help you search |
| So | 因果、转折 | So I'll navigate to... |
| I'll | 直接动作 | I'll just type the text |
| Let me try | 尝试语气 | Let me try extracting... |
| I'm going to | 即将动作 | I'm going to click this |

## 动词的多样性

### 中文动词变化

**"打开"的替换**:
- 打开
- 访问
- 进入
- 跳转到
- 看看
- 去看

**"点击"的替换**:
- 点击
- 点一下
- 按一下
- 点这个
- 戳一下（口语化）

**"输入"的替换**:
- 输入
- 填入
- 写入
- 打字
- 填写

**"抓取"的替换**:
- 抓取
- 获取
- 提取
- 拿到
- 采集

### 英文动词变化

**"open"的替换**:
- open
- navigate to
- go to
- visit
- check out

**"click"的替换**:
- click
- tap
- press
- hit
- select

**"type"的替换**:
- type
- enter
- input
- write
- fill in

**"extract"的替换**:
- extract
- grab
- get
- pull
- retrieve
- fetch

## 测试建议

### 测试不同场景

1. **连续操作**:
   - 用户: "打开百度"
   - 用户: "搜索内容"
   - 用户: "点击第一个结果"
   - **期望**: 每次回复句式不同

2. **相似操作**:
   - 用户: "打开淘宝"
   - 用户: "打开京东"
   - **期望**: 即使操作类似，表达也应该有变化

3. **不同工具**:
   - browser_navigate
   - browser_click
   - browser_type
   - browser_extract
   - **期望**: 不同工具用不同的表达方式

### 观察指标

✅ **好的表现**:
- 每次回复的开头不同
- 使用了不同的语气词
- 句式结构有变化
- 动词使用多样化

❌ **需要改进**:
- 连续回复句式完全相同
- 总是用同一个语气词
- 没有使用语气词（太正式）
- 表达过于机械

## Prompt Engineering 技巧

### 1. 提供多样化的例子

**不好的做法**:
```
Example: 我将使用 tool_name 做某事。
```

**好的做法**:
```
Examples (各种不同风格):
- 好的，让我...
- 嗯，我来...
- 那我先...
- 让我试试...
```

### 2. 明确禁止固定模板

**关键语句**:
```
Don't always use the same sentence structure.
USE AS INSPIRATION, NOT TEMPLATES
```

### 3. 鼓励创造性

**关键语句**:
```
Be creative and natural!
Vary your expressions
```

### 4. 提供具体指导

**具体的风格提示**:
- 列出可用的语气词
- 展示不同的句式
- 说明对话的自然性

## 相关文件

### 修改的文件

- **backend/agent/tools/init.go**
  - `instructionsDescription` 常量（第 108-135 行）
  - 大幅扩展和改进了提示词

## 注意事项

### 1. LLM 模型的影响

- **Claude**: 通常能很好地理解和执行多样化指令
- **GPT-4**: 也能较好地产生变化
- **较小的模型**: 可能还是会比较固定

### 2. 温度参数

适当提高 temperature 可以增加回复的多样性：
- `temperature: 0.7-0.9` → 更多样化
- `temperature: 0.0-0.3` → 更固定

### 3. 一致性与多样性的平衡

- ✅ 保持：语言匹配、说明目的
- ✅ 变化：句式、语气词、动词选择
- ⚖️ 平衡：既自然又专业

## 未来改进

### 可能的增强

1. **根据用户风格调整**:
   - 检测用户的说话风格
   - 匹配用户的正式程度

2. **上下文感知**:
   - 连续对话时避免重复句式
   - 记住之前使用过的表达

3. **更多语言支持**:
   - 添加更多语言的例子
   - 不同语言的语气词

## 总结

### ✅ 完成的工作

1. 添加创造性和自然性的明确要求
2. 提供具体的风格提示（语气词、句式）
3. 扩充例子数量（从2个到8个）
4. 强调例子是启发性的，不是模板
5. 展示多种句式和表达方式

### 📊 改进效果

| 指标 | 修改前 | 修改后 |
|------|--------|--------|
| 句式多样性 | ⚠️ 固定 | ✅ 灵活多变 |
| 语气词使用 | ❌ 无 | ✅ 丰富 |
| 自然度 | 😐 机械 | 😊 自然 |
| 对话感 | ⚠️ 正式 | ✅ 亲切 |

### 🎯 用户体验提升

**修改前**:
```
用户: "打开这个网页"
AI: "我将使用 browser_navigate 工具打开这个网页，来获取页面的最新内容。"
用户: "点击按钮"
AI: "我将使用 browser_click 工具点击这个按钮，来获取页面的最新内容。"
用户: 😐 怎么每次都一样？
```

**修改后**:
```
用户: "打开这个网页"
AI: "好的，让我用 browser_navigate 打开这个页面看看。"
用户: "点击按钮"
AI: "嗯，我来点击一下这个按钮。"
用户: "输入搜索内容"
AI: "那我先在这里输入你要搜的内容。"
用户: 😊 很自然，像在和真人对话！
```

现在 AI 的工具调用说明更加灵活和自然，不会像机器人一样重复固定句式了！🎉
