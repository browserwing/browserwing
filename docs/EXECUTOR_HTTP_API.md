# Executor HTTP API 文档

## 概述

Executor HTTP API 提供了一组 RESTful 接口，用于通过 HTTP 请求控制浏览器和执行自动化操作。这些接口可以被外部应用、CI/CD 系统、Claude Skills 等调用。

**基础 URL**: `http://<host>/api/v1/executor`

**认证方式**: JWT Token 或 API Key

- **JWT Token**: 在 `Authorization` header 中使用 `Bearer <token>`
- **API Key**: 在 `X-BrowserWing-Key` header 中传递

## API 端点列表

### 帮助和导出

#### 0. 获取帮助信息
```http
GET /api/v1/executor/help
```

获取所有可用命令的详细信息，包括参数、示例、工作流建议。

**可选参数**: `?command=<name>` - 查询特定命令的详情

**用途**: Claude 可以通过此接口自动发现所有可用命令

---

#### 0.5. 导出 Claude Skills
```http
GET /api/v1/executor/export/skill
```

一键导出完整的 Claude Skills SKILL.md 文件，包含所有 API 的使用说明。

**响应**: Markdown 文件下载

**文件名**: `EXECUTOR_SKILL_<timestamp>.md`

**cURL 示例**:
```bash
curl -X GET 'http://localhost:8080/api/v1/executor/export/skill' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -o EXECUTOR_SKILL.md
```

**用途**: 
- 快速生成 Claude Skills 文档
- 在 Claude 中导入后立即使用
- 无需手动编写 Skill 配置

详细说明: [EXECUTOR_SKILL_EXPORT.md](./EXECUTOR_SKILL_EXPORT.md)

---

### 页面导航和操作

#### 1. 导航到 URL
```http
POST /api/v1/executor/navigate
```

**请求体**:
```json
{
  "url": "https://example.com",
  "wait_until": "load",  // 可选: "load", "domcontentloaded", "networkidle"
  "timeout": 60          // 可选: 超时时间（秒），默认 60
}
```

**响应**:
```json
{
  "success": true,
  "message": "Successfully navigated to https://example.com",
  "timestamp": "2026-01-15T10:30:00Z",
  "data": {
    "url": "https://example.com",
    "semantic_tree": "Button [1]: Login\nInput Element [1]: Email\n..."
  }
}
```

**cURL 示例**:
```bash
curl -X POST 'http://localhost:8080/api/v1/executor/navigate' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -H 'Content-Type: application/json' \
  -d '{
    "url": "https://example.com",
    "wait_until": "load"
  }'
```

---

#### 2. 点击元素
```http
POST /api/v1/executor/click
```

**请求体**:
```json
{
  "identifier": "#login-button",  // CSS selector, XPath, label, 或语义树索引
  "wait_visible": true,            // 可选: 等待元素可见
  "wait_enabled": true,            // 可选: 等待元素可用
  "timeout": 10,                   // 可选: 超时时间（秒）
  "button": "left",                // 可选: "left", "right", "middle"
  "click_count": 1                 // 可选: 点击次数
}
```

**支持的 identifier 格式**:
- CSS Selector: `#login-button`, `.submit-btn`, `button[type="submit"]`
- XPath: `//button[@id='login']`
- 文本内容: `Login` (会查找包含此文本的 button 或 a 标签)
- ARIA Label: 会自动查找 `aria-label` 属性
- 语义树索引: `Clickable Element [1]`, `[1]`, `clickable [1]`

**响应**:
```json
{
  "success": true,
  "message": "Successfully clicked element: #login-button",
  "timestamp": "2026-01-15T10:30:00Z",
  "data": {
    "semantic_tree": "..."
  }
}
```

**cURL 示例**:
```bash
curl -X POST 'http://localhost:8080/api/v1/executor/click' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -H 'Content-Type: application/json' \
  -d '{
    "identifier": "#login-button",
    "wait_visible": true
  }'
```

---

#### 3. 输入文本
```http
POST /api/v1/executor/type
```

**请求体**:
```json
{
  "identifier": "#email-input",
  "text": "user@example.com",
  "clear": true,           // 可选: 清空现有内容
  "wait_visible": true,    // 可选: 等待元素可见
  "timeout": 10,           // 可选: 超时时间（秒）
  "delay": 0               // 可选: 每个字符输入延迟（毫秒）
}
```

**响应**:
```json
{
  "success": true,
  "message": "Successfully typed into element: #email-input",
  "timestamp": "2026-01-15T10:30:00Z",
  "data": {
    "text": "user@example.com"
  }
}
```

---

#### 4. 选择下拉框选项
```http
POST /api/v1/executor/select
```

**请求体**:
```json
{
  "identifier": "#country-select",
  "value": "United States",
  "wait_visible": true,
  "timeout": 10
}
```

---

#### 5. 鼠标悬停
```http
POST /api/v1/executor/hover
```

**请求体**:
```json
{
  "identifier": ".dropdown-trigger",
  "wait_visible": true,
  "timeout": 10
}
```

---

#### 6. 等待元素
```http
POST /api/v1/executor/wait
```

**请求体**:
```json
{
  "identifier": "#loading-spinner",
  "state": "hidden",  // "visible", "hidden", "enabled"
  "timeout": 30       // 秒
}
```

---

#### 7. 滚动到底部
```http
POST /api/v1/executor/scroll-to-bottom
```

**无需请求体**

**响应**:
```json
{
  "success": true,
  "message": "Successfully scrolled to bottom",
  "timestamp": "2026-01-15T10:30:00Z"
}
```

---

#### 8. 后退
```http
POST /api/v1/executor/go-back
```

---

#### 9. 前进
```http
POST /api/v1/executor/go-forward
```

---

#### 10. 刷新页面
```http
POST /api/v1/executor/reload
```

---

#### 11. 按键
```http
POST /api/v1/executor/press-key
```

**请求体**:
```json
{
  "key": "Enter",  // "Enter", "Tab", "Escape", "Backspace", "ArrowDown", 等
  "ctrl": false,   // 可选: Ctrl 键
  "shift": false,  // 可选: Shift 键
  "alt": false,    // 可选: Alt 键
  "meta": false    // 可选: Meta/Command 键
}
```

**支持的按键**:
- `Enter`, `Return`
- `Tab`
- `Escape`, `Esc`
- `Backspace`
- `Delete`
- `ArrowUp`, `ArrowDown`, `ArrowLeft`, `ArrowRight`
- `Home`, `End`
- `PageUp`, `PageDown`
- `Space`
- 单个字符: `a`, `b`, `1`, `@`, 等

**cURL 示例**:
```bash
# 按 Enter 键
curl -X POST 'http://localhost:8080/api/v1/executor/press-key' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -H 'Content-Type: application/json' \
  -d '{
    "key": "Enter"
  }'

# 按 Ctrl+S
curl -X POST 'http://localhost:8080/api/v1/executor/press-key' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -H 'Content-Type: application/json' \
  -d '{
    "key": "s",
    "ctrl": true
  }'
```

---

#### 12. 调整窗口大小
```http
POST /api/v1/executor/resize
```

**请求体**:
```json
{
  "width": 1920,
  "height": 1080
}
```

---

### 数据提取和获取

#### 13. 获取元素文本
```http
POST /api/v1/executor/get-text
```

**请求体**:
```json
{
  "identifier": "h1"
}
```

**响应**:
```json
{
  "success": true,
  "message": "Successfully retrieved text",
  "timestamp": "2026-01-15T10:30:00Z",
  "data": {
    "text": "Welcome to Our Website"
  }
}
```

---

#### 14. 获取元素值
```http
POST /api/v1/executor/get-value
```

**请求体**:
```json
{
  "identifier": "#email-input"
}
```

**响应**:
```json
{
  "success": true,
  "message": "Successfully retrieved value",
  "timestamp": "2026-01-15T10:30:00Z",
  "data": {
    "value": "user@example.com"
  }
}
```

---

#### 15. 提取数据
```http
POST /api/v1/executor/extract
```

**请求体**:
```json
{
  "selector": ".product-item",
  "type": "text",          // "text", "html", "attribute", "property"
  "attr": "href",          // 当 type 为 "attribute" 或 "property" 时使用
  "fields": ["text", "href", "src"],  // 要提取的字段列表
  "multiple": true         // 是否提取多个元素
}
```

**响应示例（multiple: true）**:
```json
{
  "success": true,
  "message": "Successfully extracted data",
  "timestamp": "2026-01-15T10:30:00Z",
  "data": {
    "result": [
      {
        "text": "Product 1",
        "href": "/product/1"
      },
      {
        "text": "Product 2",
        "href": "/product/2"
      }
    ]
  }
}
```

**cURL 示例**:
```bash
# 提取所有产品标题
curl -X POST 'http://localhost:8080/api/v1/executor/extract' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -H 'Content-Type: application/json' \
  -d '{
    "selector": ".product-title",
    "type": "text",
    "multiple": true
  }'

# 提取单个元素的多个属性
curl -X POST 'http://localhost:8080/api/v1/executor/extract' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -H 'Content-Type: application/json' \
  -d '{
    "selector": ".main-image",
    "fields": ["src", "alt"],
    "multiple": false
  }'
```

---

#### 16. 获取页面信息
```http
GET /api/v1/executor/page-info
```

**响应**:
```json
{
  "success": true,
  "message": "Successfully retrieved page info",
  "timestamp": "2026-01-15T10:30:00Z",
  "data": {
    "url": "https://example.com/page",
    "title": "Page Title"
  }
}
```

---

#### 17. 获取页面内容（HTML）
```http
GET /api/v1/executor/page-content
```

**响应**:
```json
{
  "success": true,
  "message": "Successfully retrieved page content",
  "timestamp": "2026-01-15T10:30:00Z",
  "data": {
    "html": "<!DOCTYPE html><html>...</html>"
  }
}
```

---

#### 18. 获取页面文本
```http
GET /api/v1/executor/page-text
```

**响应**:
```json
{
  "success": true,
  "message": "Successfully retrieved page text",
  "timestamp": "2026-01-15T10:30:00Z",
  "data": {
    "text": "Welcome to our website..."
  }
}
```

---

### 语义树和元素查找

#### 19. 获取语义树
```http
GET /api/v1/executor/semantic-tree
```

**响应**:
```json
{
  "success": true,
  "tree": {
    "elements": [
      {
        "id": 1,
        "role": "button",
        "name": "Login",
        "selector": "#login-btn",
        "bounds": {"x": 100, "y": 200, "width": 80, "height": 30}
      }
    ]
  },
  "tree_text": "Clickable Element [1]: Login\nInput Element [1]: Email\n..."
}
```

**用途**:
- 了解页面上有哪些可交互的元素
- 获取元素的语义树索引用于后续操作
- 查看元素的角色、名称和位置

---

#### 20. 获取可点击元素
```http
GET /api/v1/executor/clickable-elements
```

**响应**:
```json
{
  "success": true,
  "elements": [
    {
      "id": 1,
      "role": "button",
      "name": "Login",
      "selector": "#login-btn"
    },
    {
      "id": 2,
      "role": "link",
      "name": "Sign Up",
      "selector": "a.signup-link"
    }
  ],
  "count": 2
}
```

---

#### 21. 获取输入元素
```http
GET /api/v1/executor/input-elements
```

**响应**:
```json
{
  "success": true,
  "elements": [
    {
      "id": 1,
      "role": "textbox",
      "name": "Email",
      "selector": "#email-input"
    },
    {
      "id": 2,
      "role": "textbox",
      "name": "Password",
      "selector": "#password-input"
    }
  ],
  "count": 2
}
```

---

### 高级功能

#### 22. 截图
```http
POST /api/v1/executor/screenshot
```

**请求体**:
```json
{
  "full_page": false,  // 是否截取整个页面
  "quality": 80,       // 图片质量 1-100
  "format": "png"      // "png" 或 "jpeg"
}
```

**响应**:
```json
{
  "success": true,
  "message": "Successfully captured screenshot (12345 bytes)",
  "timestamp": "2026-01-15T10:30:00Z",
  "data": {
    "data": "<base64-encoded-image>",
    "format": "png",
    "size": 12345
  }
}
```

---

#### 23. 执行 JavaScript
```http
POST /api/v1/executor/evaluate
```

**请求体**:
```json
{
  "script": "() => document.title"
}
```

**响应**:
```json
{
  "success": true,
  "message": "Successfully executed script",
  "timestamp": "2026-01-15T10:30:00Z",
  "data": {
    "result": "Page Title"
  }
}
```

**cURL 示例**:
```bash
# 获取页面标题
curl -X POST 'http://localhost:8080/api/v1/executor/evaluate' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -H 'Content-Type: application/json' \
  -d '{
    "script": "() => document.title"
  }'

# 获取所有链接
curl -X POST 'http://localhost:8080/api/v1/executor/evaluate' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -H 'Content-Type: application/json' \
  -d '{
    "script": "() => Array.from(document.querySelectorAll(\"a\")).map(a => a.href)"
  }'
```

---

#### 24. 批量执行操作
```http
POST /api/v1/executor/batch
```

**请求体**:
```json
{
  "operations": [
    {
      "type": "navigate",
      "params": {
        "url": "https://example.com"
      },
      "stop_on_error": true
    },
    {
      "type": "click",
      "params": {
        "identifier": "#login-button"
      },
      "stop_on_error": true
    },
    {
      "type": "type",
      "params": {
        "identifier": "#email",
        "text": "user@example.com"
      },
      "stop_on_error": true
    }
  ]
}
```

**支持的操作类型**:
- `navigate`: 导航
- `click`: 点击
- `type`: 输入
- `select`: 选择
- `wait`: 等待

**响应**:
```json
{
  "operations": [
    {
      "success": true,
      "message": "Successfully navigated to https://example.com",
      "timestamp": "2026-01-15T10:30:00Z"
    },
    {
      "success": true,
      "message": "Successfully clicked element: #login-button",
      "timestamp": "2026-01-15T10:30:05Z"
    }
  ],
  "success": 2,
  "failed": 0,
  "start_time": "2026-01-15T10:30:00Z",
  "end_time": "2026-01-15T10:30:05Z",
  "duration": "5s"
}
```

---

## 完整使用示例

### 示例 1: 登录网站

```bash
#!/bin/bash

API_KEY="your-api-key"
BASE_URL="http://localhost:8080/api/v1/executor"

# 1. 导航到登录页面
curl -X POST "$BASE_URL/navigate" \
  -H "X-BrowserWing-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/login"}'

# 2. 输入用户名
curl -X POST "$BASE_URL/type" \
  -H "X-BrowserWing-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "identifier": "#username",
    "text": "myusername"
  }'

# 3. 输入密码
curl -X POST "$BASE_URL/type" \
  -H "X-BrowserWing-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "identifier": "#password",
    "text": "mypassword"
  }'

# 4. 点击登录按钮
curl -X POST "$BASE_URL/click" \
  -H "X-BrowserWing-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "identifier": "#login-button",
    "wait_visible": true
  }'

# 5. 等待登录成功（等待欢迎消息出现）
curl -X POST "$BASE_URL/wait" \
  -H "X-BrowserWing-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "identifier": ".welcome-message",
    "state": "visible",
    "timeout": 10
  }'
```

---

### 示例 2: 搜索和提取数据

```bash
#!/bin/bash

API_KEY="your-api-key"
BASE_URL="http://localhost:8080/api/v1/executor"

# 1. 导航到搜索页面
curl -X POST "$BASE_URL/navigate" \
  -H "X-BrowserWing-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/search"}'

# 2. 输入搜索关键词
curl -X POST "$BASE_URL/type" \
  -H "X-BrowserWing-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "identifier": "#search-input",
    "text": "laptop"
  }'

# 3. 按 Enter 键提交搜索
curl -X POST "$BASE_URL/press-key" \
  -H "X-BrowserWing-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"key": "Enter"}'

# 4. 等待搜索结果加载
curl -X POST "$BASE_URL/wait" \
  -H "X-BrowserWing-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "identifier": ".search-results",
    "state": "visible",
    "timeout": 10
  }'

# 5. 提取搜索结果
curl -X POST "$BASE_URL/extract" \
  -H "X-BrowserWing-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "selector": ".product-item",
    "fields": ["text", "href"],
    "multiple": true
  }'
```

---

### 示例 3: 使用批量操作

```bash
#!/bin/bash

API_KEY="your-api-key"
BASE_URL="http://localhost:8080/api/v1/executor"

curl -X POST "$BASE_URL/batch" \
  -H "X-BrowserWing-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "operations": [
      {
        "type": "navigate",
        "params": {
          "url": "https://example.com/form"
        },
        "stop_on_error": true
      },
      {
        "type": "type",
        "params": {
          "identifier": "#name",
          "text": "John Doe"
        },
        "stop_on_error": true
      },
      {
        "type": "type",
        "params": {
          "identifier": "#email",
          "text": "john@example.com"
        },
        "stop_on_error": true
      },
      {
        "type": "select",
        "params": {
          "identifier": "#country",
          "value": "United States"
        },
        "stop_on_error": true
      },
      {
        "type": "click",
        "params": {
          "identifier": "#submit-button"
        },
        "stop_on_error": true
      }
    ]
  }'
```

---

## Claude Skills 集成

### 创建 Claude Skill

你可以创建一个 Claude Skill 来使用这些 API：

**SKILL.md**:
```markdown
---
name: browserwing-executor
description: Control browser automation through HTTP API. Supports navigation, clicking, typing, data extraction, and more.
---

# BrowserWing Executor Skill

This skill provides browser automation capabilities through HTTP APIs.

## API Base URL

`http://<host>/api/v1/executor`

## Authentication

Use API Key authentication:
```bash
X-BrowserWing-Key: your-api-key-here
```

## Available Operations

### Navigate to URL
```bash
POST /navigate
{
  "url": "https://example.com"
}
```

### Click Element
```bash
POST /click
{
  "identifier": "#button-id"
}
```

### Type Text
```bash
POST /type
{
  "identifier": "#input-id",
  "text": "Hello World"
}
```

### Extract Data
```bash
POST /extract
{
  "selector": ".data-item",
  "fields": ["text", "href"],
  "multiple": true
}
```

### Get Page Info
```bash
GET /page-info
```

### Get Semantic Tree
```bash
GET /semantic-tree
```

## Instructions

When user asks to automate browser tasks:

1. **Understand the task**: Break down what needs to be done
2. **Get page structure**: Use `/semantic-tree` or `/clickable-elements` to understand the page
3. **Execute operations**: Use appropriate endpoints to perform actions
4. **Extract data**: Use `/extract` to get data from the page
5. **Return results**: Present extracted data or operation results to the user

## Example Workflow

User: "Search for 'laptop' on example.com and get the first 5 results"

Your actions:
1. Call `POST /navigate` with `{"url": "https://example.com"}`
2. Call `POST /type` to enter search term
3. Call `POST /press-key` with `{"key": "Enter"}`
4. Call `POST /wait` to wait for results
5. Call `POST /extract` to get product data
6. Present results to user

## Tips

- Always check page structure first with `/semantic-tree`
- Use semantic tree indices like `[1]`, `[2]` for element identification
- Handle errors gracefully and explain what went wrong
- For complex workflows, use `/batch` endpoint
```

---

## 错误处理

所有错误响应的格式：

```json
{
  "error": "error.operationFailed",
  "detail": "Detailed error message"
}
```

**常见错误码**:
- `error.invalidRequest`: 请求参数无效
- `error.navigationFailed`: 导航失败
- `error.clickFailed`: 点击失败
- `error.typeFailed`: 输入失败
- `error.elementNotFound`: 元素未找到
- `error.timeout`: 操作超时
- `error.unauthorized`: 认证失败

---

## 最佳实践

### 1. 使用语义树索引

语义树索引是一种更可靠的元素定位方式：

```bash
# 首先获取语义树
curl -X GET 'http://localhost:8080/api/v1/executor/semantic-tree' \
  -H 'X-BrowserWing-Key: your-api-key'

# 响应中会显示:
# Clickable Element [1]: Login Button
# Input Element [1]: Email
# Input Element [2]: Password

# 然后使用索引进行操作
curl -X POST 'http://localhost:8080/api/v1/executor/click' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -H 'Content-Type: application/json' \
  -d '{"identifier": "Clickable Element [1]"}'
```

### 2. 等待元素加载

在动态页面上，确保元素加载完成：

```bash
# 先等待元素可见
curl -X POST 'http://localhost:8080/api/v1/executor/wait' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -H 'Content-Type: application/json' \
  -d '{
    "identifier": "#dynamic-content",
    "state": "visible",
    "timeout": 10
  }'

# 然后再进行操作
curl -X POST 'http://localhost:8080/api/v1/executor/click' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -H 'Content-Type: application/json' \
  -d '{"identifier": "#dynamic-content"}'
```

### 3. 使用批量操作提高效率

对于多个连续操作，使用批量接口：

```bash
curl -X POST 'http://localhost:8080/api/v1/executor/batch' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -H 'Content-Type: application/json' \
  -d '{
    "operations": [
      {"type": "navigate", "params": {"url": "..."}, "stop_on_error": true},
      {"type": "type", "params": {"identifier": "...", "text": "..."}, "stop_on_error": true},
      {"type": "click", "params": {"identifier": "..."}, "stop_on_error": true}
    ]
  }'
```

### 4. 超时设置

为不同的操作设置合适的超时：
- **快速操作** (click, type): 5-10 秒
- **页面导航**: 30-60 秒
- **数据加载**: 10-30 秒

### 5. 错误重试

对于不稳定的操作，实现重试逻辑：

```bash
#!/bin/bash

MAX_RETRIES=3
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
  RESPONSE=$(curl -X POST 'http://localhost:8080/api/v1/executor/click' \
    -H 'X-BrowserWing-Key: your-api-key' \
    -H 'Content-Type: application/json' \
    -d '{"identifier": "#button"}')
  
  if echo "$RESPONSE" | grep -q '"success":true'; then
    echo "Success!"
    break
  fi
  
  RETRY_COUNT=$((RETRY_COUNT + 1))
  echo "Retry $RETRY_COUNT/$MAX_RETRIES..."
  sleep 2
done
```

---

## 性能考虑

### 1. 减少不必要的语义树提取

Navigate 和 Click 操作会自动返回语义树，不需要额外调用 `/semantic-tree`。

### 2. 并发请求

这些 API 目前不支持并发操作同一个浏览器实例。如需并发，请使用多个浏览器实例。

### 3. 资源清理

长时间运行后，建议定期重启浏览器：

```bash
# 停止浏览器
curl -X POST 'http://localhost:8080/api/v1/browser/stop' \
  -H 'X-BrowserWing-Key: your-api-key'

# 启动浏览器
curl -X POST 'http://localhost:8080/api/v1/browser/start' \
  -H 'X-BrowserWing-Key: your-api-key'
```

---

## 相关文档

- [Executor MCP Tools](./executor/README.md) - Executor 的 MCP 工具文档
- [Semantic Tree Documentation](./SEMANTIC_RECORDING_ENHANCEMENT.md) - 语义树详细文档
- [API Authentication](./API_AUTHENTICATION.md) - API 认证文档

---

## 总结

Executor HTTP API 提供了完整的浏览器自动化能力，适合：

✅ **外部应用集成**: 通过 HTTP 调用控制浏览器
✅ **CI/CD 自动化**: 在测试流程中使用
✅ **Claude Skills**: 让 Claude AI 能够操作浏览器
✅ **Webhook 触发**: 基于事件的自动化
✅ **定时任务**: 通过 cron 执行自动化脚本

所有接口都支持 JWT Token 和 API Key 认证，确保安全访问！
