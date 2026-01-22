# BrowserWing 与 Playwright-MCP 命令对齐

## 概述

本文档说明 BrowserWing Executor 如何对齐 playwright-mcp 的命令规范，提供一致的 MCP 工具能力。

## 命令映射表

### ✅ 已实现（需重命名或优化）

| Playwright-MCP | BrowserWing 当前 | 新名称 | 状态 |
|----------------|-----------------|--------|------|
| `browser_navigate` | Navigate | Navigate | ✅ 保持 |
| `browser_navigate_back` | GoBack | NavigateBack | 🔄 重命名 |
| `browser_click` | Click | Click | ✅ 保持 |
| `browser_hover` | Hover | Hover | ✅ 保持 |
| `browser_drag` | Drag | Drag | ✅ 保持 |
| `browser_type` | Type | Type | ✅ 保持 |
| `browser_press_key` | PressKey | PressKey | ✅ 保持 |
| `browser_select_option` | Select | SelectOption | 🔄 重命名 |
| `browser_file_upload` | FileUpload | FileUpload | ✅ 保持 |
| `browser_resize` | Resize | Resize | ✅ 保持 |
| `browser_close` | ClosePage | Close | 🔄 重命名 |
| `browser_console_messages` | GetConsoleMessages | GetConsoleMessages | ✅ 保持 |
| `browser_network_requests` | GetNetworkRequests | GetNetworkRequests | ✅ 保持 |
| `browser_snapshot` | GetSemanticTree | **GetAccessibilitySnapshot** | 🔄 **重要改名** |
| `browser_take_screenshot` | Screenshot | TakeScreenshot | 🔄 重命名 |
| `browser_evaluate` | Evaluate | Evaluate | ✅ 保持 |
| `browser_handle_dialog` | HandleDialog | HandleDialog | ✅ 保持 |
| `browser_wait_for` | WaitFor | WaitFor | ✅ 保持 |

### ⚠️ 需要补充的命令

| Playwright-MCP | 说明 | 优先级 |
|----------------|------|--------|
| `browser_install` | 安装/配置浏览器 | P2 (可选) |
| `browser_tabs` | 标签页管理 (list/new/switch/close) | P0 (必需) |
| `browser_fill_form` | 批量填写表单字段 | P1 (重要) |
| `browser_run_code` | 运行 Playwright 代码片段 | P2 (高级) |

### 🎯 BrowserWing 独有功能（保留）

| BrowserWing 命令 | 说明 | 保留原因 |
|-----------------|------|----------|
| `Extract` | 批量数据提取 | 核心能力，与 snapshot 互补 |
| `GetText` | 获取元素文本 | 基础工具 |
| `GetValue` | 获取表单值 | 基础工具 |
| `ScrollToBottom` | 滚动到底部 | 常用操作 |
| `GoForward` | 前进 | 浏览器基础功能 |
| `Reload` | 刷新页面 | 浏览器基础功能 |

## 核心改动：Semantic Tree → Accessibility Snapshot

### 1. 概念对齐

**之前：** `GetSemanticTree()` - 返回"语义树"
**现在：** `GetAccessibilitySnapshot()` - 返回"可访问性快照"

这更符合：
- Web 标准术语（Accessibility Tree）
- Playwright/Puppeteer 的命名
- playwright-mcp 的概念

### 2. 返回格式

保持现有的树状结构，但更新命名和文档：

```go
// AccessibilitySnapshot 可访问性快照
type AccessibilitySnapshot struct {
    Elements     map[string]*AccessibilityNode  // 节点索引
    AXNodeMap    map[proto.AccessibilityAXNodeID]*proto.AccessibilityAXNode
    BackendIDMap map[proto.DOMBackendNodeID]*AccessibilityNode
}

// AccessibilityNode 可访问性节点
type AccessibilityNode struct {
    ID             string                    // 语义 ID (如 "button_0", "input_0")
    Role           string                    // ARIA role
    Name           string                    // 可访问名称
    Description    string                    // 描述
    Value          string                    // 值（表单元素）
    BackendNodeID  proto.DOMBackendNodeID   // 用于定位元素
    // ... 其他字段
}
```

### 3. 使用场景对比

#### Playwright-MCP 的 browser_snapshot

```typescript
// 获取页面的可访问性快照，供 LLM 理解页面结构
const snapshot = await browser_snapshot();

// 返回简化的树状结构
{
  role: 'WebArea',
  name: 'Example Page',
  children: [
    { role: 'heading', name: 'Welcome', level: 1 },
    { role: 'button', name: 'Click Me' },
    { role: 'textbox', name: 'Email', value: '' }
  ]
}
```

#### BrowserWing 的 GetAccessibilitySnapshot

```go
// 获取可访问性快照
snapshot, err := executor.GetAccessibilitySnapshot(ctx)

// 返回索引化的节点结构（更适合程序处理）
{
  "elements": {
    "button_0": {
      "id": "button_0",
      "role": "button",
      "name": "Click Me",
      "backendNodeID": 123
    },
    "input_0": {
      "id": "input_0",
      "role": "textbox",
      "name": "Email",
      "value": ""
    }
  }
}
```

## 新增命令实现

### 1. browser_tabs - 标签页管理 (P0)

```go
// TabsOptions 标签页操作选项
type TabsOptions struct {
    Action string `json:"action"` // list, new, switch, close
    TabID  string `json:"tab_id,omitempty"` // 用于 switch/close
    URL    string `json:"url,omitempty"`    // 用于 new
}

// Tabs 管理浏览器标签页
func (e *Executor) Tabs(ctx context.Context, opts *TabsOptions) (*OperationResult, error) {
    page := e.GetRodPage()
    if page == nil {
        return nil, fmt.Errorf("no active page")
    }
    
    browser := page.Browser()
    
    switch opts.Action {
    case "list":
        pages, err := browser.Pages()
        // 返回所有标签页信息
        
    case "new":
        newPage := browser.MustPage(opts.URL)
        // 创建新标签页
        
    case "switch":
        // 切换到指定标签页
        
    case "close":
        // 关闭指定标签页
    }
}
```

### 2. browser_fill_form - 批量填表单 (P1)

```go
// FillFormOptions 批量填表单选项
type FillFormOptions struct {
    Fields map[string]string `json:"fields"` // 字段选择器 -> 值
}

// FillForm 批量填写表单字段
func (e *Executor) FillForm(ctx context.Context, opts *FillFormOptions) (*OperationResult, error) {
    page := e.GetRodPage()
    if page == nil {
        return nil, fmt.Errorf("no active page")
    }
    
    results := make(map[string]interface{})
    
    for selector, value := range opts.Fields {
        elem, err := page.Element(selector)
        if err != nil {
            results[selector] = map[string]interface{}{
                "success": false,
                "error": err.Error(),
            }
            continue
        }
        
        // 清空并输入新值
        elem.MustSelectAllText().MustInput(value)
        
        results[selector] = map[string]interface{}{
            "success": true,
            "value": value,
        }
    }
    
    return &OperationResult{
        Success:   true,
        Message:   fmt.Sprintf("Filled %d form fields", len(opts.Fields)),
        Timestamp: time.Now(),
        Data: map[string]interface{}{
            "results": results,
        },
    }, nil
}
```

## MCP 工具注册更新

### 旧的注册方式

```go
server.AddTool(mcpgo.Tool{
    Name: "get_semantic_tree",
    Description: "获取页面的语义树结构",
    // ...
})
```

### 新的注册方式

```go
server.AddTool(mcpgo.Tool{
    Name: "browser_snapshot",
    Description: "Get the accessibility snapshot of the current page. Returns a tree structure representing the page's accessibility tree, which is cleaner than raw DOM and better for LLMs to understand.",
    InputSchema: mcpgo.ToolInputSchema{
        Type: "object",
        Properties: map[string]interface{}{
            "max_depth": {
                "type": "number",
                "description": "Maximum depth of the tree (default: unlimited)",
            },
        },
    },
})

server.AddTool(mcpgo.Tool{
    Name: "browser_tabs",
    Description: "Manage browser tabs (list, create, switch, close)",
    InputSchema: mcpgo.ToolInputSchema{
        Type: "object",
        Properties: map[string]interface{}{
            "action": {
                "type": "string",
                "enum": []string{"list", "new", "switch", "close"},
                "description": "Action to perform",
            },
            "tab_id": {
                "type": "string",
                "description": "Tab ID (for switch/close actions)",
            },
            "url": {
                "type": "string",
                "description": "URL to open (for new action)",
            },
        },
        Required: []string{"action"},
    },
})

server.AddTool(mcpgo.Tool{
    Name: "browser_fill_form",
    Description: "Fill multiple form fields at once",
    InputSchema: mcpgo.ToolInputSchema{
        Type: "object",
        Properties: map[string]interface{}{
            "fields": {
                "type": "object",
                "description": "Map of CSS selectors to values",
                "additionalProperties": {
                    "type": "string",
                },
            },
        },
        Required: []string{"fields"},
    },
})
```

## 文档更新

### SKILL.md 更新

```markdown
## Accessibility Snapshot

Get the accessibility tree of the current page. The accessibility tree is a simplified representation of the page structure, more suitable for LLMs to understand than raw DOM.

**Command:** `browser_snapshot`

**Parameters:**
- `max_depth` (optional): Maximum tree depth

**Response:**
```json
{
  "success": true,
  "data": {
    "elements": {
      "button_0": {
        "role": "button",
        "name": "Submit",
        "clickable": true
      },
      "input_0": {
        "role": "textbox",
        "name": "Email Address",
        "value": "",
        "required": true
      }
    }
  }
}
```

**Use Cases:**
- Understanding page structure
- Finding interactive elements
- Generating element selectors for automation

## Tab Management

Manage browser tabs.

**Command:** `browser_tabs`

**Actions:**

1. **List all tabs:**
```json
{ "action": "list" }
```

2. **Create new tab:**
```json
{ "action": "new", "url": "https://example.com" }
```

3. **Switch to tab:**
```json
{ "action": "switch", "tab_id": "tab-123" }
```

4. **Close tab:**
```json
{ "action": "close", "tab_id": "tab-123" }
```

## Fill Form

Fill multiple form fields at once.

**Command:** `browser_fill_form`

**Example:**
```json
{
  "fields": {
    "input[name='email']": "user@example.com",
    "input[name='password']": "secret123",
    "select[name='country']": "US"
  }
}
```
```

## 实施计划

### Phase 1: 核心重命名 (立即)
- [x] 重命名 `SemanticTree` → `AccessibilitySnapshot`
- [x] 重命名 `SemanticNode` → `AccessibilityNode`
- [x] 更新 MCP 工具注册名称：`get_semantic_tree` → `browser_snapshot`
- [x] 更新文档中的所有引用

### Phase 2: 补充缺失命令 (P0)
- [ ] 实现 `Tabs()` - 标签页管理
- [ ] 注册 `browser_tabs` MCP 工具
- [ ] 添加相关测试

### Phase 3: 优化现有命令 (P1)
- [ ] 实现 `FillForm()` - 批量填表单
- [ ] 重命名方法以保持一致性
- [ ] 更新所有文档和示例

### Phase 4: 高级功能 (P2)
- [ ] 实现 `browser_run_code` (可选)
- [ ] 实现 `browser_install` (可选)

## 迁移指南

### 对于现有用户

如果你在使用 `get_semantic_tree`，请更新为 `browser_snapshot`：

**旧代码：**
```python
result = execute_mcp_tool("get_semantic_tree", {})
```

**新代码：**
```python
result = execute_mcp_tool("browser_snapshot", {})
```

返回的数据结构保持兼容，只是命名更新为 accessibility 相关术语。

## 对齐收益

1. **标准化** - 与 Web 标准术语一致
2. **互操作性** - 更容易与其他工具集成
3. **易理解** - Accessibility Tree 是通用概念
4. **功能完整** - 补齐 playwright-mcp 的核心能力
5. **文档统一** - 减少学习成本

## 相关文件

- `backend/executor/semantic.go` → 重命名为 `accessibility.go`
- `backend/executor/operations.go` - 补充新命令
- `backend/executor/mcp_tools.go` - 更新工具注册
- `SKILL.md` - 更新文档

## 参考资料

- [Playwright Accessibility API](https://playwright.dev/docs/accessibility-testing)
- [playwright-mcp Commands](https://github.com/microsoft/playwright-mcp)
- [WAI-ARIA Accessibility Tree](https://www.w3.org/TR/wai-aria-1.2/#accessibility_tree)
