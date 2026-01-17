# Executor Help API

## 概述

Executor Help API 提供了一个自助服务接口，让 Claude 或其他客户端可以自动发现所有可用的浏览器自动化命令及其使用方法。

**端点**: `GET /api/v1/executor/help`

## 功能

### 1. 获取所有可用命令

**请求**:
```bash
curl -X GET 'http://localhost:8080/api/v1/executor/help' \
  -H 'X-BrowserWing-Key: your-api-key'
```

**响应**:
```json
{
  "total_commands": 23,
  "base_url": "/api/v1/executor",
  "authentication": {
    "methods": ["JWT Token", "API Key"],
    "jwt": "Authorization: Bearer <token>",
    "api_key": "X-BrowserWing-Key: <api-key>"
  },
  "workflow": [
    "1. Call GET /semantic-tree to understand page structure",
    "2. Use element indices ([1], [2]) or CSS selectors for operations",
    "3. Call appropriate operation endpoints (navigate, click, type, etc.)",
    "4. Extract data using /extract endpoint",
    "5. Use /batch for multiple operations"
  ],
  "element_identifiers": {
    "css_selector": "#id, .class, button[type='submit']",
    "xpath": "//button[@id='login']",
    "text_content": "Login, Sign Up (will find button/link with this text)",
    "semantic_index": "[1], Clickable Element [1], Input Element [2]",
    "aria_label": "Searches for elements with aria-label attribute",
    "recommendation": "Use semantic-tree first to get element indices"
  },
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
          "description": "Target URL to navigate to",
          "example": "https://example.com"
        },
        "wait_until": {
          "type": "string",
          "required": false,
          "description": "Wait condition: load, domcontentloaded, networkidle",
          "default": "load"
        },
        "timeout": {
          "type": "number",
          "required": false,
          "description": "Timeout in seconds",
          "default": 60
        }
      },
      "example": {
        "url": "https://example.com",
        "wait_until": "load"
      },
      "returns": "Operation result with semantic tree"
    },
    {
      "name": "click",
      "method": "POST",
      "endpoint": "/api/v1/executor/click",
      "description": "Click an element on the page",
      "parameters": {
        "identifier": {
          "type": "string",
          "required": true,
          "description": "Element identifier: CSS selector, XPath, text, semantic index ([1], Clickable Element [1])",
          "example": "#button-id or [1]"
        },
        "wait_visible": {
          "type": "boolean",
          "required": false,
          "description": "Wait for element to be visible",
          "default": true
        }
      },
      "example": {
        "identifier": "#login-button",
        "wait_visible": true
      }
    }
    // ... 更多命令
  ],
  "examples": {
    "simple_workflow": {
      "description": "Navigate and click a button",
      "steps": [
        {
          "step": 1,
          "action": "Navigate",
          "endpoint": "POST /navigate",
          "payload": {"url": "https://example.com"}
        },
        {
          "step": 2,
          "action": "Get page structure",
          "endpoint": "GET /semantic-tree"
        },
        {
          "step": 3,
          "action": "Click button",
          "endpoint": "POST /click",
          "payload": {"identifier": "[1]"}
        }
      ]
    },
    "data_extraction": {
      "description": "Search and extract results",
      "steps": [
        {
          "step": 1,
          "endpoint": "POST /navigate",
          "payload": {"url": "https://example.com/search"}
        },
        {
          "step": 2,
          "endpoint": "POST /type",
          "payload": {"identifier": "#search", "text": "query"}
        },
        {
          "step": 3,
          "endpoint": "POST /press-key",
          "payload": {"key": "Enter"}
        },
        {
          "step": 4,
          "endpoint": "POST /wait",
          "payload": {"identifier": ".results", "state": "visible"}
        },
        {
          "step": 5,
          "endpoint": "POST /extract",
          "payload": {
            "selector": ".item",
            "fields": ["text", "href"],
            "multiple": true
          }
        }
      ]
    }
  }
}
```

---

### 2. 查询特定命令的详细信息

**请求**:
```bash
curl -X GET 'http://localhost:8080/api/v1/executor/help?command=click' \
  -H 'X-BrowserWing-Key: your-api-key'
```

**响应**:
```json
{
  "command": {
    "name": "click",
    "method": "POST",
    "endpoint": "/api/v1/executor/click",
    "description": "Click an element on the page",
    "parameters": {
      "identifier": {
        "type": "string",
        "required": true,
        "description": "Element identifier: CSS selector, XPath, text, semantic index ([1], Clickable Element [1])",
        "example": "#button-id or [1]"
      },
      "wait_visible": {
        "type": "boolean",
        "required": false,
        "description": "Wait for element to be visible",
        "default": true
      },
      "timeout": {
        "type": "number",
        "required": false,
        "description": "Timeout in seconds",
        "default": 10
      }
    },
    "example": {
      "identifier": "#login-button",
      "wait_visible": true
    },
    "returns": "Operation result with updated semantic tree"
  }
}
```

---

## 包含的命令列表

### 页面导航和操作
1. **navigate** - 导航到 URL
2. **click** - 点击元素
3. **type** - 输入文本
4. **select** - 选择下拉框选项
5. **hover** - 鼠标悬停
6. **wait** - 等待元素状态
7. **press-key** - 按键
8. **scroll-to-bottom** - 滚动到底部
9. **go-back** - 后退
10. **go-forward** - 前进
11. **reload** - 刷新页面

### 数据提取
12. **get-text** - 获取元素文本
13. **get-value** - 获取输入值
14. **extract** - 提取数据
15. **page-info** - 获取页面信息
16. **page-text** - 获取页面文本
17. **page-content** - 获取页面 HTML

### 页面分析
18. **semantic-tree** - 获取语义树（推荐首先调用）
19. **clickable-elements** - 获取所有可点击元素
20. **input-elements** - 获取所有输入元素

### 高级功能
21. **screenshot** - 截图
22. **evaluate** - 执行 JavaScript
23. **batch** - 批量执行操作

---

## Claude Skills 集成示例

### 在 SKILL.md 中使用

```markdown
---
name: browserpilot-automation
description: Browser automation through HTTP API with self-discovery
---

# BrowserPilot Automation Skill

## Discovery

First, discover available commands:

```bash
GET /api/v1/executor/help
```

This returns:
- All available commands
- Parameter specifications
- Usage examples
- Workflow recommendations

## Typical Workflow

1. **Discovery**: Call `/help` to understand available operations
2. **Structure**: Call `/semantic-tree` to understand page structure  
3. **Identify**: Use semantic indices like `[1]`, `[2]` to reference elements
4. **Execute**: Call appropriate operations (click, type, extract, etc.)
5. **Extract**: Use `/extract` to get data from the page

## Instructions

When user requests browser automation:

1. If unsure about available operations, call `GET /help`
2. If unsure about specific command, call `GET /help?command=<name>`
3. Always call `/semantic-tree` first to understand page structure
4. Use element indices from semantic tree for reliable operations
5. Handle errors gracefully and explain to user

## Example

User: "Go to example.com and click the login button"

Your actions:
1. (Optional) Call GET /help to refresh command knowledge
2. Call POST /navigate with {"url": "https://example.com"}
3. Call GET /semantic-tree to see page structure
4. Response shows: "Clickable Element [1]: Login"
5. Call POST /click with {"identifier": "[1]"}
6. Report success to user
```

---

## Python 客户端示例

```python
import requests

class BrowserPilotClient:
    def __init__(self, base_url, api_key):
        self.base_url = base_url
        self.headers = {'X-BrowserWing-Key': api_key}
        self.commands = None
    
    def discover_commands(self):
        """发现所有可用命令"""
        response = requests.get(
            f'{self.base_url}/api/v1/executor/help',
            headers=self.headers
        )
        self.commands = response.json()
        return self.commands
    
    def get_command_help(self, command_name):
        """获取特定命令的帮助"""
        response = requests.get(
            f'{self.base_url}/api/v1/executor/help',
            params={'command': command_name},
            headers=self.headers
        )
        return response.json()
    
    def list_commands(self):
        """列出所有命令名称"""
        if not self.commands:
            self.discover_commands()
        return [cmd['name'] for cmd in self.commands['commands']]
    
    def execute_command(self, command_name, **params):
        """执行命令"""
        if not self.commands:
            self.discover_commands()
        
        # 查找命令配置
        cmd_config = next(
            (c for c in self.commands['commands'] if c['name'] == command_name),
            None
        )
        
        if not cmd_config:
            raise ValueError(f"Unknown command: {command_name}")
        
        # 执行请求
        method = cmd_config['method']
        endpoint = cmd_config['endpoint']
        
        if method == 'GET':
            response = requests.get(
                f'{self.base_url}{endpoint}',
                headers=self.headers
            )
        else:
            response = requests.post(
                f'{self.base_url}{endpoint}',
                json=params,
                headers=self.headers
            )
        
        return response.json()

# 使用示例
client = BrowserPilotClient('http://localhost:8080', 'your-api-key')

# 发现命令
commands = client.discover_commands()
print(f"Total commands: {commands['total_commands']}")

# 列出所有命令
print("Available commands:", client.list_commands())

# 获取特定命令的帮助
help_info = client.get_command_help('click')
print("Click command:", help_info)

# 执行命令
result = client.execute_command('navigate', url='https://example.com')
print("Navigation result:", result)

result = client.execute_command('click', identifier='[1]')
print("Click result:", result)
```

---

## JavaScript/Node.js 客户端示例

```javascript
const axios = require('axios');

class BrowserPilotClient {
  constructor(baseUrl, apiKey) {
    this.baseUrl = baseUrl;
    this.headers = { 'X-BrowserWing-Key': apiKey };
    this.commands = null;
  }

  async discoverCommands() {
    const response = await axios.get(
      `${this.baseUrl}/api/v1/executor/help`,
      { headers: this.headers }
    );
    this.commands = response.data;
    return this.commands;
  }

  async getCommandHelp(commandName) {
    const response = await axios.get(
      `${this.baseUrl}/api/v1/executor/help`,
      {
        params: { command: commandName },
        headers: this.headers
      }
    );
    return response.data;
  }

  listCommands() {
    if (!this.commands) {
      throw new Error('Call discoverCommands() first');
    }
    return this.commands.commands.map(c => c.name);
  }

  async executeCommand(commandName, params = {}) {
    if (!this.commands) {
      await this.discoverCommands();
    }

    const cmdConfig = this.commands.commands.find(
      c => c.name === commandName
    );

    if (!cmdConfig) {
      throw new Error(`Unknown command: ${commandName}`);
    }

    const config = {
      method: cmdConfig.method.toLowerCase(),
      url: `${this.baseUrl}${cmdConfig.endpoint}`,
      headers: this.headers
    };

    if (cmdConfig.method === 'POST') {
      config.data = params;
    }

    const response = await axios(config);
    return response.data;
  }
}

// 使用示例
(async () => {
  const client = new BrowserPilotClient(
    'http://localhost:8080',
    'your-api-key'
  );

  // 发现命令
  const commands = await client.discoverCommands();
  console.log(`Total commands: ${commands.total_commands}`);

  // 列出所有命令
  console.log('Available commands:', client.listCommands());

  // 获取特定命令的帮助
  const helpInfo = await client.getCommandHelp('extract');
  console.log('Extract command:', helpInfo);

  // 执行自动化
  await client.executeCommand('navigate', {
    url: 'https://example.com'
  });

  const tree = await client.executeCommand('semantic-tree');
  console.log('Page structure:', tree);

  await client.executeCommand('click', { identifier: '[1]' });
})();
```

---

## 响应格式说明

### 命令对象结构

每个命令包含以下字段：

```json
{
  "name": "命令名称",
  "method": "HTTP方法 (GET/POST)",
  "endpoint": "API端点路径",
  "description": "命令描述",
  "parameters": {
    "param_name": {
      "type": "参数类型",
      "required": true/false,
      "description": "参数说明",
      "example": "示例值",
      "default": "默认值 (可选)"
    }
  },
  "example": {
    // 请求示例
  },
  "returns": "返回值说明",
  "note": "额外说明 (可选)"
}
```

### Workflow 指导

`workflow` 字段提供了推荐的使用流程：

1. 使用 `/semantic-tree` 了解页面结构
2. 使用元素索引或 CSS 选择器
3. 调用适当的操作端点
4. 使用 `/extract` 提取数据
5. 使用 `/batch` 执行批量操作

### Element Identifiers 说明

`element_identifiers` 字段说明了所有支持的元素定位方式：

- **CSS Selector**: 标准 CSS 选择器
- **XPath**: XPath 表达式
- **Text Content**: 元素的文本内容
- **Semantic Index**: 语义树索引（推荐）
- **ARIA Label**: aria-label 属性

---

## 优势

### 🔍 自动发现
- Claude 或其他客户端无需事先知道所有命令
- 可以动态学习新增的功能
- 减少文档维护负担

### 📖 内置文档
- 每个命令都有完整的参数说明
- 包含使用示例
- 提供工作流指导

### 🎯 类型安全
- 明确的参数类型和要求
- 默认值说明
- 返回值说明

### 💡 最佳实践
- 内置工作流推荐
- 元素定位方式说明
- 示例工作流展示

---

## 与 Claude 的对话示例

### 示例 1: 自动发现

```
用户: 帮我使用 BrowserPilot 自动化浏览器

Claude: 让我先看看有哪些可用的操作...

[调用 GET /api/v1/executor/help]

我发现了 23 个可用命令，包括：
- 页面导航: navigate, go-back, go-forward, reload
- 元素交互: click, type, select, hover
- 数据提取: extract, get-text, get-value
- 页面分析: semantic-tree, clickable-elements, input-elements
- 高级功能: screenshot, evaluate, batch

你想做什么操作呢？
```

### 示例 2: 命令详情查询

```
用户: extract 命令怎么用？

Claude: 让我查看 extract 命令的详细说明...

[调用 GET /api/v1/executor/help?command=extract]

extract 命令用于从页面元素提取数据，参数包括：

必需参数:
- selector: CSS 选择器，指定要提取的元素

可选参数:
- type: 提取类型（text, html, attribute, property）
- fields: 要提取的字段列表（如 ["text", "href", "src"]）
- multiple: 是否提取多个元素（默认 false）

示例用法:
{
  "selector": ".product-item",
  "fields": ["text", "href"],
  "multiple": true
}

这会提取所有 .product-item 元素的文本和链接。
```

### 示例 3: 完整自动化流程

```
用户: 在 example.com 搜索 laptop 并提取结果

Claude: 
1. 首先导航到网站
   [POST /navigate {"url": "https://example.com"}]

2. 获取页面结构了解元素位置
   [GET /semantic-tree]
   
   响应显示:
   - Input Element [1]: Search Box
   - Clickable Element [1]: Search Button

3. 在搜索框输入 "laptop"
   [POST /type {"identifier": "Input Element [1]", "text": "laptop"}]

4. 按 Enter 提交搜索
   [POST /press-key {"key": "Enter"}]

5. 等待搜索结果加载
   [POST /wait {"identifier": ".results", "state": "visible", "timeout": 10}]

6. 提取搜索结果
   [POST /extract {
     "selector": ".search-result",
     "fields": ["text", "href"],
     "multiple": true
   }]

找到 15 个结果：
1. Gaming Laptop - $1299 (link)
2. Business Laptop - $899 (link)
...
```

---

## 总结

Help API 提供了：

✅ **自动发现**: 无需预先知道所有命令
✅ **完整文档**: 每个命令的详细说明和示例
✅ **类型安全**: 明确的参数类型和要求
✅ **最佳实践**: 内置工作流指导和推荐
✅ **Claude 友好**: 专为 AI Agent 设计的自助服务接口

通过这个接口，Claude 可以：
- 🔍 自动发现所有可用操作
- 📖 查询任何命令的详细用法
- 🎯 了解推荐的工作流
- 💡 学习元素定位的最佳方式
- 🚀 快速上手浏览器自动化

**端点**: `GET /api/v1/executor/help` 或 `GET /api/v1/executor/help?command=<name>`

**认证**: `X-BrowserWing-Key: <api-key>` 或 `Authorization: Bearer <token>`
