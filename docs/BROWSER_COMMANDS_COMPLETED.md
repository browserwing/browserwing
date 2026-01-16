# Browser 命令完善文档

## 概述

本文档记录了新增和完善的浏览器自动化命令。所有命令已成功实现并集成到 MCP 工具系统中。

## 新增命令列表

### 1. 截图命令 (browser_take_screenshot)

**功能**: 对当前页面进行截图

**分类**: Capture（捕获）

**参数**:
- `full_page` (boolean, 可选): 是否捕获完整页面，默认 false
- `format` (string, 可选): 图片格式，支持 png 或 jpeg，默认 png

**实现**:
```go
func (e *Executor) Screenshot(ctx context.Context, opts *ScreenshotOptions) (*OperationResult, error)
```

**示例**:
```json
{
  "tool": "browser_take_screenshot",
  "arguments": {
    "full_page": true,
    "format": "png"
  }
}
```

---

### 2. 执行脚本命令 (browser_evaluate)

**功能**: 在浏览器上下文中执行 JavaScript 代码

**分类**: Scripting（脚本）

**参数**:
- `script` (string, 必需): 要执行的 JavaScript 代码

**实现**:
```go
func (e *Executor) Evaluate(ctx context.Context, script string) (*OperationResult, error)
```

**示例**:
```json
{
  "tool": "browser_evaluate",
  "arguments": {
    "script": "document.title"
  }
}
```

**使用场景**:
- 获取页面信息
- 修改页面元素
- 执行复杂的 DOM 操作
- 注入自定义 JavaScript 代码

---

### 3. 按键命令 (browser_press_key)

**功能**: 模拟键盘按键

**分类**: Interaction（交互）

**参数**:
- `key` (string, 必需): 要按的键（Enter, Tab, ArrowUp, Escape 等）
- `ctrl` (boolean, 可选): 是否同时按下 Ctrl 键
- `shift` (boolean, 可选): 是否同时按下 Shift 键
- `alt` (boolean, 可选): 是否同时按下 Alt 键
- `meta` (boolean, 可选): 是否同时按下 Meta 键（Mac 的 Command 或 Windows 键）

**实现**:
```go
func (e *Executor) PressKey(ctx context.Context, key string, opts *PressKeyOptions) (*OperationResult, error)
```

**支持的按键**:
- 特殊键: Enter, Tab, Escape, Backspace, Delete, Space
- 方向键: ArrowUp, ArrowDown, ArrowLeft, ArrowRight
- 导航键: Home, End, PageUp, PageDown
- 单个字符: a-z, 0-9 等

**示例**:
```json
{
  "tool": "browser_press_key",
  "arguments": {
    "key": "Enter"
  }
}
```

**组合键示例**:
```json
{
  "tool": "browser_press_key",
  "arguments": {
    "key": "s",
    "ctrl": true
  }
}
```

---

### 4. 调整窗口命令 (browser_resize)

**功能**: 调整浏览器窗口大小

**分类**: Window（窗口）

**参数**:
- `width` (number, 必需): 窗口宽度（像素）
- `height` (number, 必需): 窗口高度（像素）

**实现**:
```go
func (e *Executor) Resize(ctx context.Context, width, height int) (*OperationResult, error)
```

**示例**:
```json
{
  "tool": "browser_resize",
  "arguments": {
    "width": 1920,
    "height": 1080
  }
}
```

**使用场景**:
- 测试响应式设计
- 模拟不同设备的视口
- 截图前调整窗口大小

---

### 5. 拖拽命令 (browser_drag)

**功能**: 将一个元素拖拽到另一个元素

**分类**: Interaction（交互）

**参数**:
- `from_identifier` (string, 必需): 源元素标识符
- `to_identifier` (string, 必需): 目标元素标识符

**实现**:
```go
func (e *Executor) Drag(ctx context.Context, fromIdentifier, toIdentifier string) (*OperationResult, error)
```

**示例**:
```json
{
  "tool": "browser_drag",
  "arguments": {
    "from_identifier": "Clickable Element [1]",
    "to_identifier": "Clickable Element [5]"
  }
}
```

**使用场景**:
- 拖放操作
- 重新排序列表项
- 移动可拖动的元素

---

### 6. 关闭页面命令 (browser_close)

**功能**: 关闭当前浏览器页面/标签页

**分类**: Window（窗口）

**参数**: 无

**实现**:
```go
func (e *Executor) ClosePage(ctx context.Context) (*OperationResult, error)
```

**示例**:
```json
{
  "tool": "browser_close",
  "arguments": {}
}
```

---

### 7. 文件上传命令 (browser_file_upload)

**功能**: 上传文件到文件输入元素

**分类**: Interaction（交互）

**参数**:
- `identifier` (string, 必需): 文件输入元素标识符
- `file_paths` (array, 必需): 要上传的文件路径数组

**实现**:
```go
func (e *Executor) FileUpload(ctx context.Context, identifier string, filePaths []string) (*OperationResult, error)
```

**示例**:
```json
{
  "tool": "browser_file_upload",
  "arguments": {
    "identifier": "Input Element [1]",
    "file_paths": ["/path/to/file1.jpg", "/path/to/file2.pdf"]
  }
}
```

**使用场景**:
- 上传图片
- 上传文档
- 批量上传文件

---

### 8. 对话框处理命令 (browser_handle_dialog)

**功能**: 配置如何处理 JavaScript 对话框（alert, confirm, prompt）

**分类**: Dialog（对话框）

**参数**:
- `accept` (boolean, 必需): 是否接受对话框
- `text` (string, 可选): 对于 prompt 对话框，要输入的文本

**实现**:
```go
func (e *Executor) HandleDialog(ctx context.Context, accept bool, text string) (*OperationResult, error)
```

**示例 - 接受 confirm 对话框**:
```json
{
  "tool": "browser_handle_dialog",
  "arguments": {
    "accept": true
  }
}
```

**示例 - 拒绝 alert 对话框**:
```json
{
  "tool": "browser_handle_dialog",
  "arguments": {
    "accept": false
  }
}
```

**示例 - 输入 prompt 文本**:
```json
{
  "tool": "browser_handle_dialog",
  "arguments": {
    "accept": true,
    "text": "My Input"
  }
}
```

---

### 9. 控制台消息命令 (browser_console_messages)

**功能**: 获取浏览器控制台消息

**分类**: Debug（调试）

**参数**: 无

**实现**:
```go
func (e *Executor) GetConsoleMessages(ctx context.Context) (*OperationResult, error)
```

**示例**:
```json
{
  "tool": "browser_console_messages",
  "arguments": {}
}
```

**返回数据**:
```json
{
  "messages": [
    {
      "type": "log",
      "timestamp": "2026-01-15T12:34:56Z",
      "args": ["Hello", "World"]
    },
    {
      "type": "error",
      "timestamp": "2026-01-15T12:35:00Z",
      "args": ["Error occurred"]
    }
  ]
}
```

**使用场景**:
- 调试 JavaScript 错误
- 监控页面日志
- 检查警告信息

---

### 10. 网络请求命令 (browser_network_requests)

**功能**: 获取页面发出的网络请求

**分类**: Debug（调试）

**参数**: 无

**实现**:
```go
func (e *Executor) GetNetworkRequests(ctx context.Context) (*OperationResult, error)
```

**示例**:
```json
{
  "tool": "browser_network_requests",
  "arguments": {}
}
```

**返回数据**:
```json
{
  "requests": [
    {
      "url": "https://example.com/api/data",
      "method": "GET",
      "timestamp": "2026-01-15T12:34:56Z",
      "type": "XHR"
    },
    {
      "url": "https://example.com/image.png",
      "method": "GET",
      "timestamp": "2026-01-15T12:35:00Z",
      "type": "Image"
    }
  ]
}
```

**使用场景**:
- 监控 API 调用
- 调试网络问题
- 分析页面加载性能

---

## 命令分类总览

### Navigation (导航)
- `browser_navigate` - 导航到 URL
- `browser_scroll` - 滚动页面

### Interaction (交互)
- `browser_click` - 点击元素
- `browser_type` - 输入文本
- `browser_select` - 选择下拉选项
- `browser_press_key` - 按键 ✨ 新增
- `browser_drag` - 拖拽元素 ✨ 新增
- `browser_file_upload` - 上传文件 ✨ 新增

### Capture (捕获)
- `browser_take_screenshot` - 截图 ✨ 新增

### Scripting (脚本)
- `browser_evaluate` - 执行 JavaScript ✨ 新增

### Data (数据)
- `browser_extract` - 提取数据

### Analysis (分析)
- `browser_get_semantic_tree` - 获取语义树
- `browser_get_page_info` - 获取页面信息

### Synchronization (同步)
- `browser_wait_for` - 等待元素状态

### Window (窗口)
- `browser_resize` - 调整窗口大小 ✨ 新增
- `browser_close` - 关闭页面 ✨ 新增

### Dialog (对话框)
- `browser_handle_dialog` - 处理对话框 ✨ 新增

### Debug (调试)
- `browser_console_messages` - 获取控制台消息 ✨ 新增
- `browser_network_requests` - 获取网络请求 ✨ 新增

---

## 技术实现细节

### 新增数据类型

**PressKeyOptions** (`types.go`):
```go
type PressKeyOptions struct {
    Ctrl  bool // Ctrl 键
    Shift bool // Shift 键
    Alt   bool // Alt 键
    Meta  bool // Meta 键
}
```

### 代码结构

1. **operations.go** - 实现核心操作函数
   - Screenshot
   - Evaluate
   - PressKey
   - Resize
   - Drag
   - ClosePage
   - FileUpload
   - HandleDialog
   - GetConsoleMessages
   - GetNetworkRequests

2. **mcp_tools.go** - 注册 MCP 工具
   - registerScreenshotTool
   - registerEvaluateTool
   - registerPressKeyTool
   - registerResizeTool
   - registerDragTool
   - registerClosePageTool
   - registerFileUploadTool
   - registerHandleDialogTool
   - registerGetConsoleMessagesTool
   - registerGetNetworkRequestsTool

3. **types.go** - 定义数据类型
   - PressKeyOptions

### 工具元数据

所有新增工具都已添加到 `GetExecutorToolsMetadata()` 函数中，包含：
- 工具名称
- 描述
- 分类
- 参数定义

这些元数据用于：
- 前端工具管理页面展示
- API 文档生成
- 工具配置初始化

---

## 测试建议

### 1. 截图测试
```bash
# 测试完整页面截图
curl -X POST http://localhost:8080/api/mcp/call \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "browser_take_screenshot",
    "arguments": {
      "full_page": true,
      "format": "png"
    }
  }'
```

### 2. JavaScript 执行测试
```bash
# 测试执行脚本
curl -X POST http://localhost:8080/api/mcp/call \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "browser_evaluate",
    "arguments": {
      "script": "document.querySelectorAll('a').length"
    }
  }'
```

### 3. 按键测试
```bash
# 测试按 Enter 键
curl -X POST http://localhost:8080/api/mcp/call \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "browser_press_key",
    "arguments": {
      "key": "Enter"
    }
  }'
```

### 4. 拖拽测试
```bash
# 测试拖拽
curl -X POST http://localhost:8080/api/mcp/call \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "browser_drag",
    "arguments": {
      "from_identifier": "Clickable Element [1]",
      "to_identifier": "Clickable Element [2]"
    }
  }'
```

---

## 使用示例场景

### 场景 1: 表单自动填写和提交
```javascript
// 1. 导航到表单页面
browser_navigate({ url: "https://example.com/form" })

// 2. 填写表单字段
browser_type({ identifier: "Input Element [1]", text: "John Doe" })
browser_type({ identifier: "Input Element [2]", text: "john@example.com" })

// 3. 选择下拉选项
browser_select({ identifier: "Input Element [3]", value: "Option 1" })

// 4. 按 Tab 键移动到下一个字段
browser_press_key({ key: "Tab" })

// 5. 按 Enter 提交表单
browser_press_key({ key: "Enter" })

// 6. 等待成功消息
browser_wait_for({ identifier: ".success-message", state: "visible" })

// 7. 截图保存结果
browser_take_screenshot({ full_page: true })
```

### 场景 2: 网页调试和诊断
```javascript
// 1. 导航到页面
browser_navigate({ url: "https://example.com" })

// 2. 执行脚本检查页面状态
browser_evaluate({ script: "window.performance.timing" })

// 3. 获取控制台错误
browser_console_messages({})

// 4. 获取网络请求
browser_network_requests({})

// 5. 调整窗口大小测试响应式
browser_resize({ width: 768, height: 1024 })

// 6. 截图
browser_take_screenshot({ full_page: true })
```

### 场景 3: 文件上传流程
```javascript
// 1. 导航到上传页面
browser_navigate({ url: "https://example.com/upload" })

// 2. 点击上传按钮
browser_click({ identifier: "Clickable Element [1]" })

// 3. 上传文件
browser_file_upload({
  identifier: "Input Element [1]",
  file_paths: ["/path/to/document.pdf"]
})

// 4. 处理确认对话框
browser_handle_dialog({ accept: true })

// 5. 等待上传完成
browser_wait_for({ identifier: ".upload-success", state: "visible" })
```

### 场景 4: 拖放排序
```javascript
// 1. 导航到页面
browser_navigate({ url: "https://example.com/sortable-list" })

// 2. 获取语义树查看可拖动元素
browser_get_semantic_tree({ simple: true })

// 3. 拖动第一项到第三项位置
browser_drag({
  from_identifier: "Clickable Element [1]",
  to_identifier: "Clickable Element [3]"
})

// 4. 截图验证结果
browser_take_screenshot({ full_page: false })
```

---

## 注意事项

### 1. 异步事件处理
- `GetConsoleMessages` 和 `GetNetworkRequests` 使用事件监听器
- 当前实现会等待 100ms 收集消息
- 对于实时监控，可能需要扩展实现

### 2. 对话框处理
- `HandleDialog` 需要在对话框出现**之前**调用
- 它设置了一个处理器来自动响应对话框
- 一次只能设置一个处理器

### 3. 文件上传
- 文件路径必须是服务器上可访问的绝对路径
- 支持批量上传（传入多个文件路径）
- 元素必须是 `<input type="file">` 类型

### 4. 拖拽操作
- 需要元素可见且可交互
- 计算元素中心点进行拖拽
- 对于复杂的拖拽（如自定义拖放库），可能需要使用 `browser_evaluate`

### 5. 按键操作
- 修饰键（Ctrl, Shift, Alt, Meta）会在主键按下期间保持
- 单个字符可以直接使用
- 特殊键需要使用预定义的键名

---

## 工具配置集成

所有新增的 Executor 工具都已集成到工具配置系统中：

1. **自动初始化**: 服务启动时自动创建工具配置
2. **工具类型**: 标记为 `ToolTypePreset`（预设工具）
3. **默认启用**: 所有新工具默认启用
4. **前端管理**: 可以在前端工具管理页面查看和管理

配置结构：
```json
{
  "id": "browser_take_screenshot",
  "name": "browser_take_screenshot",
  "type": "preset",
  "description": "Take a screenshot of the current page",
  "enabled": true,
  "parameters": {
    "category": "Capture"
  }
}
```

---

## 总结

本次完善新增了 10 个浏览器自动化命令，覆盖了：

✅ **截图** - 页面捕获功能
✅ **脚本执行** - JavaScript 代码执行
✅ **按键模拟** - 键盘操作
✅ **窗口管理** - 大小调整和关闭
✅ **拖拽操作** - 元素拖放
✅ **文件上传** - 文件选择和上传
✅ **对话框处理** - alert/confirm/prompt 处理
✅ **调试工具** - 控制台和网络监控

这些命令与现有的 10 个命令一起，构成了一个**完整的浏览器自动化工具集**，能够满足绝大多数网页自动化场景的需求。

**总计**: 20 个浏览器自动化命令
- 10 个原有命令
- 10 个新增命令 ✨

所有命令都已：
- ✅ 实现核心功能
- ✅ 注册到 MCP 服务
- ✅ 集成到工具配置系统
- ✅ 添加元数据和文档
- ✅ 编译通过
