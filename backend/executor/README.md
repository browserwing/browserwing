# Executor 模块

Executor 模块提供通用的浏览器自动化能力，参考 [agent-browser](https://github.com/vercel-labs/agent-browser) 设计，提供语义化的浏览器操作接口。

## 核心功能

### 1. 语义树提取 (Semantic Tree)
自动提取页面中所有可交互元素的语义信息，包括：
- 按钮、链接等可点击元素
- 输入框、文本域等表单元素
- 下拉选择框
- 元素的标签、占位符、值等语义信息
- 元素的位置、可见性、可用性等状态

### 2. 智能元素查找
支持多种方式查找元素：
- CSS 选择器
- XPath
- 元素标签/文本
- ARIA 标签
- 占位符文本

### 3. 丰富的操作方法
- **导航**: `Navigate`, `GoBack`, `GoForward`, `Reload`
- **交互**: `Click`, `Type`, `Select`, `Hover`
- **等待**: `WaitFor`, `WaitUntilReady`
- **数据提取**: `Extract`, `GetText`, `GetValue`
- **截图**: `Screenshot`
- **滚动**: `ScrollToBottom`, `ScrollToElement`
- **批量操作**: `ExecuteBatch`

### 4. MCP 集成
所有操作都可以作为 MCP (Model Context Protocol) 工具注册，供外部 AI Agent 调用。

## 快速开始

### 基本用法

```go
package main

import (
    "context"
    "github.com/browserwing/browserwing/backend/executor"
    "github.com/browserwing/browserwing/services/browser"
)

func main() {
    ctx := context.Background()
    
    // 创建浏览器管理器
    browserMgr := browser.NewManager(nil, nil)
    browserMgr.Start(ctx)
    defer browserMgr.Stop()
    
    // 创建 Executor
    exec := executor.NewExecutor(browserMgr)
    
    // 导航到网页
    exec.Navigate(ctx, "https://example.com", nil)
    
    // 点击元素
    exec.Click(ctx, "button.login", nil)
    
    // 输入文本
    exec.Type(ctx, "input[name='username']", "myuser", nil)
}
```

### 使用语义树

```go
// 获取语义树
tree, _ := exec.GetSemanticTree(ctx)

// 查找所有可点击元素
clickable := tree.GetClickableElements()
for _, node := range clickable {
    fmt.Printf("Button: %s\n", node.Label)
}

// 通过标签查找元素
node := tree.FindElementByLabel("Login")
if node != nil {
    exec.Click(ctx, node.Selector, nil)
}
```

### 智能交互（通过标签）

```go
// 通过标签直接操作，无需知道具体选择器
exec.TypeByLabel(ctx, "Username", "myuser")
exec.TypeByLabel(ctx, "Password", "mypassword")
exec.ClickByLabel(ctx, "Login")
```

### 数据提取

```go
// 提取单个元素的文本
result, _ := exec.Extract(ctx, &executor.ExtractOptions{
    Selector: ".product-title",
    Type:     "text",
})

// 提取多个元素
result, _ := exec.Extract(ctx, &executor.ExtractOptions{
    Selector: ".product-item",
    Type:     "text",
    Multiple: true,
})
```

### 批量操作

```go
operations := []executor.Operation{
    {
        Type: "navigate",
        Params: map[string]interface{}{
            "url": "https://example.com",
        },
    },
    {
        Type: "type",
        Params: map[string]interface{}{
            "identifier": "input[name='search']",
            "text":       "browserwing",
        },
    },
    {
        Type: "click",
        Params: map[string]interface{}{
            "identifier": "button[type='submit']",
        },
    },
}

result, _ := exec.ExecuteBatch(ctx, operations)
```

## MCP 集成

### 注册 MCP 工具

```go
import (
    "github.com/browserwing/browserwing/backend/executor"
    "github.com/mark3labs/mcp-go/server"
)

// 创建 MCP 服务器
mcpServer := server.NewMCPServer("browserwing", "1.0.0")

// 创建 Executor
exec := executor.NewExecutor(browserMgr)

// 创建工具注册表
registry := executor.NewMCPToolRegistry(exec, mcpServer)

// 注册所有工具
registry.RegisterAllTools()
```

### 可用的 MCP 工具

#### 导航类
- `browser_navigate`: 导航到 URL
- `browser_scroll`: 滚动页面

#### 交互类
- `browser_click`: 点击元素
- `browser_type`: 输入文本
- `browser_select`: 选择下拉选项

#### 数据类
- `browser_extract`: 提取数据
- `browser_get_page_info`: 获取页面信息
- `browser_get_semantic_tree`: 获取语义树

#### 捕获类
- `browser_screenshot`: 截图

#### 同步类
- `browser_wait_for`: 等待元素状态

### MCP 工具使用示例

通过 MCP 客户端调用：

```json
{
  "tool": "browser_navigate",
  "arguments": {
    "url": "https://example.com"
  }
}
```

```json
{
  "tool": "browser_click",
  "arguments": {
    "identifier": "Login",
    "wait_visible": true
  }
}
```

```json
{
  "tool": "browser_get_semantic_tree",
  "arguments": {
    "simple": true
  }
}
```

## 核心概念

### SemanticNode (语义节点)

表示页面中的一个可交互元素：

```go
type SemanticNode struct {
    ID          string                 // 唯一标识符
    Type        string                 // 元素类型
    Role        string                 // ARIA role
    Label       string                 // 元素标签
    Placeholder string                 // 占位符
    Value       string                 // 当前值
    Text        string                 // 文本内容
    Selector    string                 // CSS 选择器
    XPath       string                 // XPath
    IsVisible   bool                   // 是否可见
    IsEnabled   bool                   // 是否可用
}
```

### OperationResult (操作结果)

所有操作都返回统一的结果结构：

```go
type OperationResult struct {
    Success   bool                   // 是否成功
    Message   string                 // 结果消息
    Data      map[string]interface{} // 返回数据
    Error     string                 // 错误信息
    Timestamp time.Time              // 时间戳
}
```

## 高级特性

### 1. 元素高亮

```go
// 高亮显示元素（用于调试）
exec.HighlightElementByLabel(ctx, "Login")
```

### 2. 可访问性评估

```go
page := exec.GetRodPage()
report, _ := executor.EvaluateAccessibility(ctx, page)
fmt.Printf("Found %d accessibility issues\n", report.TotalIssues)
```

### 3. 自定义等待条件

```go
exec.WaitFor(ctx, ".content", &executor.WaitForOptions{
    State:   "visible",
    Timeout: 10 * time.Second,
})
```

### 4. 元素截图

```go
page := exec.GetRodPage()
elem, _ := page.Element(".target-element")
data, _ := executor.GetElementScreenshot(ctx, elem)
```

## 与 agent-browser 的对比

| 特性 | agent-browser | Executor 模块 |
|------|--------------|--------------|
| 语义树提取 | ✅ | ✅ |
| 智能元素查找 | ✅ | ✅ |
| MCP 集成 | ❌ | ✅ |
| Go 语言原生 | ❌ | ✅ |
| 批量操作 | 部分 | ✅ |
| 可访问性评估 | ❌ | ✅ |
| 自定义选项 | 有限 | 丰富 |

## 与 playwright-mcp 的对比

| 特性 | playwright-mcp | Executor 模块 |
|------|----------------|--------------|
| 底层引擎 | Playwright | Rod |
| 语义理解 | 基础 | 深度 |
| 智能查找 | 有限 | 强大 |
| 工具数量 | ~10 | 10+ |
| 批量操作 | ❌ | ✅ |
| 自定义扩展 | 困难 | 容易 |

## 设计理念

1. **语义优先**: 使用语义信息而非底层选择器进行交互
2. **智能查找**: 支持多种方式查找元素，自动选择最佳方式
3. **统一接口**: 所有操作返回统一的结果结构
4. **可扩展性**: 易于添加新的操作和工具
5. **MCP 原生**: 设计时即考虑 MCP 集成
6. **类型安全**: 充分利用 Go 的类型系统

## 示例场景

### 场景 1: 自动登录

```go
exec.Navigate(ctx, "https://example.com/login", nil)
exec.TypeByLabel(ctx, "Email", "user@example.com")
exec.TypeByLabel(ctx, "Password", "password123")
exec.ClickByLabel(ctx, "Sign In")
exec.WaitFor(ctx, ".dashboard", nil)
```

### 场景 2: 数据抓取

```go
exec.Navigate(ctx, "https://example.com/products", nil)

result, _ := exec.Extract(ctx, &executor.ExtractOptions{
    Selector: ".product",
    Multiple: true,
    Fields:   []string{"text", "href"},
})

products := result.Data["result"].([]map[string]interface{})
for _, product := range products {
    fmt.Printf("Product: %s - %s\n", product["text"], product["href"])
}
```

### 场景 3: 表单自动填写

```go
exec.Navigate(ctx, "https://example.com/form", nil)
exec.Type(ctx, "input[name='name']", "John Doe", nil)
exec.Type(ctx, "input[name='email']", "john@example.com", nil)
exec.Select(ctx, "select[name='country']", "United States", nil)
exec.Click(ctx, "input[name='agree']", nil)
exec.Click(ctx, "button[type='submit']", nil)
```

## 后续计划

- [ ] 添加更多智能操作（拖拽、双击等）
- [ ] 支持多标签页管理
- [ ] 添加性能监控和分析
- [ ] 增强错误处理和重试机制
- [ ] 添加更多 MCP 工具
- [ ] 支持浏览器扩展
- [ ] 添加视觉回归测试

## 贡献

欢迎贡献代码、报告问题或提出建议！

## 许可证

MIT License

