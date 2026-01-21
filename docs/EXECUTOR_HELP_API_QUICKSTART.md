# Executor Help API 快速入门

## 🎯 为什么需要 Help API？

在使用 Executor HTTP API 之前，你可能想知道：
- 有哪些可用的命令？
- 每个命令需要什么参数？
- 如何正确使用这些命令？

**Help API 让 Claude 和其他客户端可以自动发现和学习所有可用的操作！**

---

## 🚀 快速开始

### 1. 获取所有可用命令

```bash
curl -X GET 'http://localhost:8080/api/v1/executor/help' \
  -H 'X-BrowserWing-Key: your-api-key'
```

**返回**:
```json
{
  "total_commands": 25,
  "base_url": "/api/v1/executor",
  "authentication": {
    "methods": ["JWT Token", "API Key"],
    "api_key": "X-BrowserWing-Key: <api-key>"
  },
  "workflow": [
    "1. Call GET /semantic-tree to understand page structure",
    "2. Use element indices ([1], [2]) or CSS selectors",
    "3. Call appropriate operation endpoints",
    "4. Extract data using /extract endpoint"
  ],
  "commands": [
    {
      "name": "navigate",
      "method": "POST",
      "endpoint": "/api/v1/executor/navigate",
      "description": "Navigate to a URL",
      "parameters": {
        "url": {
          "type": "string",
          "required": true,
          "example": "https://example.com"
        }
      }
    }
    // ... 更多命令
  ]
}
```

---

### 2. 查询特定命令

```bash
curl -X GET 'http://localhost:8080/api/v1/executor/help?command=extract' \
  -H 'X-BrowserWing-Key: your-api-key'
```

**返回**:
```json
{
  "command": {
    "name": "extract",
    "method": "POST",
    "endpoint": "/api/v1/executor/extract",
    "description": "Extract data from page elements",
    "parameters": {
      "selector": {
        "type": "string",
        "required": true,
        "description": "CSS selector for elements to extract"
      },
      "fields": {
        "type": "array",
        "required": false,
        "description": "Fields to extract: text, html, href, src, value",
        "example": ["text", "href"]
      },
      "multiple": {
        "type": "boolean",
        "required": false,
        "description": "Extract multiple elements",
        "default": false
      }
    },
    "example": {
      "selector": ".product-item",
      "fields": ["text", "href"],
      "multiple": true
    }
  }
}
```

---

## 📋 完整命令列表

Help API 包含以下 25 个命令：

### 🔍 发现和帮助
- **help** - 获取所有命令的帮助信息

### 🌐 页面导航
- **navigate** - 导航到 URL
- **go-back** - 后退
- **go-forward** - 前进
- **reload** - 刷新页面

### 🖱️ 元素交互
- **click** - 点击元素
- **type** - 输入文本
- **select** - 选择下拉框
- **hover** - 鼠标悬停
- **wait** - 等待元素状态

### ⌨️ 键盘操作
- **press-key** - 按键（支持 Enter, Tab, Ctrl+S 等）

### 📊 数据提取
- **extract** - 提取页面数据
- **get-text** - 获取元素文本
- **get-value** - 获取输入值

### 📄 页面信息
- **page-info** - 获取页面 URL 和标题
- **page-text** - 获取页面所有文本
- **page-content** - 获取页面 HTML

### 🎯 页面分析（重要！）
- **semantic-tree** - 获取语义树（推荐首先调用）
- **clickable-elements** - 获取所有可点击元素
- **input-elements** - 获取所有输入元素

### 🚀 高级功能
- **screenshot** - 截图
- **evaluate** - 执行 JavaScript
- **batch** - 批量执行操作
- **scroll-to-bottom** - 滚动到底部
- **resize** - 调整窗口大小

---

## 💡 推荐工作流

### 第一步：发现命令

```bash
# Claude 或客户端首先调用
GET /api/v1/executor/help
```

这样可以：
- ✅ 了解所有可用的操作
- ✅ 查看每个操作的参数
- ✅ 看到使用示例
- ✅ 学习推荐的工作流

### 第二步：了解页面结构

```bash
# 导航到页面后，调用
GET /api/v1/executor/semantic-tree
```

返回结果会显示：
```
Clickable Element [1]: Login Button
Clickable Element [2]: Sign Up Link
Input Element [1]: Email Input
Input Element [2]: Password Input
```

### 第三步：执行操作

使用语义树索引（推荐）：
```bash
POST /api/v1/executor/click
{"identifier": "[1]"}  # 点击第一个可点击元素

POST /api/v1/executor/type
{"identifier": "Input Element [1]", "text": "user@example.com"}
```

或使用 CSS 选择器：
```bash
POST /api/v1/executor/click
{"identifier": "#login-button"}
```

---

## 🤖 Claude Skills 使用示例

### 在 SKILL.md 中使用

```markdown
---
name: browserwing-automation
description: Browser automation with self-discovery capability
---

# BrowserWing Automation Skill

## How It Works

I can control a web browser through HTTP APIs. I start by discovering what commands are available, then use them to automate tasks.

## My Workflow

1. **Discover**: I call `GET /api/v1/executor/help` to see all available commands
2. **Navigate**: I use `POST /navigate` to go to a website  
3. **Analyze**: I call `GET /semantic-tree` to understand the page structure
4. **Interact**: I use element indices like `[1]`, `[2]` to click, type, etc.
5. **Extract**: I use `POST /extract` to get data from the page

## Instructions for Me

When user asks for browser automation:

1. If I'm unsure what commands exist, call `/help`
2. If I'm unsure how a command works, call `/help?command=<name>`
3. Always call `/semantic-tree` after navigating to understand the page
4. Use semantic indices from the tree for reliable operations
5. Handle errors and explain what went wrong

## Example

User: "Search for 'laptop' on example.com"

My actions:
1. (Optional) GET /help to refresh my knowledge
2. POST /navigate {"url": "https://example.com"}  
3. GET /semantic-tree
   Response: "Input Element [1]: Search Box, Clickable Element [1]: Search Button"
4. POST /type {"identifier": "Input Element [1]", "text": "laptop"}
5. POST /press-key {"key": "Enter"}
6. POST /wait {"identifier": ".results", "state": "visible"}
7. POST /extract {"selector": ".result", "fields": ["text", "href"], "multiple": true}
8. Present results to user
```

---

## 🐍 Python 示例

```python
import requests

class BrowserWing:
    def __init__(self, base_url, api_key):
        self.base = base_url
        self.key = api_key
        self.commands = None
    
    def discover(self):
        """发现所有可用命令"""
        r = requests.get(
            f'{self.base}/api/v1/executor/help',
            headers={'X-BrowserWing-Key': self.key}
        )
        self.commands = r.json()
        print(f"✅ 发现 {self.commands['total_commands']} 个命令")
        return self.commands
    
    def help(self, command=None):
        """获取命令帮助"""
        url = f'{self.base}/api/v1/executor/help'
        params = {'command': command} if command else {}
        r = requests.get(url, 
            params=params,
            headers={'X-BrowserWing-Key': self.key}
        )
        return r.json()
    
    def run(self, command, **params):
        """执行命令"""
        cmd = self.help(command)['command']
        method = cmd['method']
        endpoint = cmd['endpoint']
        
        if method == 'GET':
            r = requests.get(
                f'{self.base}{endpoint}',
                headers={'X-BrowserWing-Key': self.key}
            )
        else:
            r = requests.post(
                f'{self.base}{endpoint}',
                json=params,
                headers={'X-BrowserWing-Key': self.key}
            )
        return r.json()

# 使用
bp = BrowserWing('http://localhost:8080', 'your-key')

# 1. 发现命令
bp.discover()

# 2. 查看特定命令
extract_help = bp.help('extract')
print(extract_help)

# 3. 执行自动化
bp.run('navigate', url='https://example.com')
tree = bp.run('semantic-tree')
print(tree['tree_text'])

bp.run('click', identifier='[1]')
data = bp.run('extract', 
    selector='.item',
    fields=['text', 'href'],
    multiple=True
)
print(data)
```

---

## 🌟 主要优势

### 1. 自动发现
- 无需预先知道所有命令
- Claude 可以动态学习新功能
- 减少文档维护

### 2. 自我解释
- 每个命令都有完整文档
- 包含参数类型和示例
- 提供使用建议

### 3. 易于集成
- 简单的 HTTP GET 请求
- 标准 JSON 响应
- 适合任何编程语言

### 4. Claude 友好
- 专为 AI Agent 设计
- 结构化的命令信息
- 内置工作流指导

---

## 📚 下一步

1. **阅读完整文档**: [EXECUTOR_HELP_API.md](./EXECUTOR_HELP_API.md)
2. **查看 API 参考**: [EXECUTOR_HTTP_API.md](./EXECUTOR_HTTP_API.md)
3. **创建 Claude Skill**: 使用上面的模板
4. **开始自动化**: 让 Claude 调用 `/help` 并开始工作！

---

## 🎉 总结

通过 Help API，你可以：

✅ **让 Claude 自动发现** 所有可用的浏览器操作
✅ **动态学习** 每个命令的使用方法
✅ **查看示例** 了解最佳实践
✅ **减少配置** 不需要预先编写大量文档

**一个 API 调用，掌握所有能力！**

```bash
curl -X GET 'http://localhost:8080/api/v1/executor/help' \
  -H 'X-BrowserWing-Key: your-api-key'
```

现在就开始让 Claude 使用 BrowserWing 进行浏览器自动化吧！🚀
