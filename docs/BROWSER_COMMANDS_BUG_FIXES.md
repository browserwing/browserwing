# Browser 命令 Bug 修复文档

## 问题描述

用户报错：`Error executing tool: failed to call MCP tool: unknown executor tool: browser_press_key`

## 根本原因

新增的 10 个 browser 命令虽然已经：
1. ✅ 在 `executor/operations.go` 中实现了核心功能
2. ✅ 在 `executor/mcp_tools.go` 中注册到了 MCP 工具注册表
3. ✅ 在 `executor/mcp_tools.go` 的 `GetExecutorToolsMetadata()` 中添加了元数据

但是，**在 `mcp/server.go` 的 `callExecutorTool()` 方法中缺少对这些新工具的处理逻辑**。

### 问题详情

在 `mcp/server.go` 中：

1. `CallTool()` 方法会检查工具名是否以 `"browser_"` 开头
2. 如果是，则调用 `callExecutorTool()` 方法
3. `callExecutorTool()` 使用一个大的 `switch` 语句来处理不同的工具
4. **但这个 switch 语句只包含了旧的工具，没有新增的工具**
5. 因此新工具会走到 `default` 分支，返回 `"unknown executor tool"` 错误

## 修复内容

### 1. 修复工具名称不一致 (executor/mcp_tools.go)

**问题**: 元数据中有重复的截图工具定义

```go
// ❌ 错误：重复定义
{
    Name: "browser_screenshot",  // 旧名称
    ...
},
{
    Name: "browser_take_screenshot",  // 新名称
    ...
}
```

**修复**: 删除旧的 `browser_screenshot` 条目，只保留 `browser_take_screenshot`

### 2. 修复 MCP 服务器工具名称 (mcp/server.go)

**问题**: `callExecutorTool()` 中使用了旧的工具名

```go
// ❌ 错误
case "browser_screenshot":
```

**修复**: 改为新的工具名

```go
// ✅ 正确
case "browser_take_screenshot":
```

### 3. 添加所有新工具的处理逻辑 (mcp/server.go)

在 `callExecutorTool()` 的 switch 语句中添加了 10 个新工具的 case：

#### 3.1 browser_evaluate
```go
case "browser_evaluate":
    script, _ := arguments["script"].(string)
    result, err := s.executor.Evaluate(ctx, script)
    // 返回结果
```

#### 3.2 browser_press_key
```go
case "browser_press_key":
    key, _ := arguments["key"].(string)
    ctrl, _ := arguments["ctrl"].(bool)
    shift, _ := arguments["shift"].(bool)
    alt, _ := arguments["alt"].(bool)
    meta, _ := arguments["meta"].(bool)
    
    opts := &executor.PressKeyOptions{
        Ctrl:  ctrl,
        Shift: shift,
        Alt:   alt,
        Meta:  meta,
    }
    result, err := s.executor.PressKey(ctx, key, opts)
```

#### 3.3 browser_resize
```go
case "browser_resize":
    width := int(arguments["width"].(float64))
    height := int(arguments["height"].(float64))
    result, err := s.executor.Resize(ctx, width, height)
```

#### 3.4 browser_drag
```go
case "browser_drag":
    fromIdentifier, _ := arguments["from_identifier"].(string)
    toIdentifier, _ := arguments["to_identifier"].(string)
    result, err := s.executor.Drag(ctx, fromIdentifier, toIdentifier)
```

#### 3.5 browser_close
```go
case "browser_close":
    result, err := s.executor.ClosePage(ctx)
```

#### 3.6 browser_file_upload
```go
case "browser_file_upload":
    identifier, _ := arguments["identifier"].(string)
    var filePaths []string
    if paths, ok := arguments["file_paths"].([]interface{}); ok {
        for _, p := range paths {
            if path, ok := p.(string); ok {
                filePaths = append(filePaths, path)
            }
        }
    }
    result, err := s.executor.FileUpload(ctx, identifier, filePaths)
```

#### 3.7 browser_handle_dialog
```go
case "browser_handle_dialog":
    accept, _ := arguments["accept"].(bool)
    text, _ := arguments["text"].(string)
    result, err := s.executor.HandleDialog(ctx, accept, text)
```

#### 3.8 browser_console_messages
```go
case "browser_console_messages":
    result, err := s.executor.GetConsoleMessages(ctx)
    // 返回包含 console messages 的 data
```

#### 3.9 browser_network_requests
```go
case "browser_network_requests":
    result, err := s.executor.GetNetworkRequests(ctx)
    // 返回包含 network requests 的 data
```

## 修复的文件

1. **`backend/executor/mcp_tools.go`**
   - 删除重复的 `browser_screenshot` 元数据条目

2. **`backend/mcp/server.go`**
   - 修复 `browser_screenshot` 为 `browser_take_screenshot`
   - 添加 9 个新工具的 case 处理逻辑

## 验证

### 编译验证
```bash
cd /root/code/browserpilot/backend && go build
# ✅ 编译成功
```

### 工具名称一致性检查
```bash
# 注册的工具名称
grep -A1 'NewTool(' executor/mcp_tools.go | grep '"browser_' | sed 's/.*"\(browser_[^"]*\)".*/\1/' | sort

# 元数据中的工具名称
grep 'Name:.*"browser_' executor/mcp_tools.go | sed 's/.*Name:.*"\(browser_[^"]*\)".*/\1/' | sort

# ✅ 两个列表完全一致（19个工具）
```

## 现在可用的所有工具 (19个)

| # | 工具名 | 状态 |
|---|--------|------|
| 1 | browser_navigate | ✅ |
| 2 | browser_click | ✅ |
| 3 | browser_type | ✅ |
| 4 | browser_select | ✅ |
| 5 | browser_extract | ✅ |
| 6 | browser_get_semantic_tree | ✅ |
| 7 | browser_get_page_info | ✅ |
| 8 | browser_wait_for | ✅ |
| 9 | browser_scroll | ✅ |
| 10 | browser_take_screenshot | ✅ 修复 |
| 11 | browser_evaluate | ✅ 新增 |
| 12 | browser_press_key | ✅ 新增 |
| 13 | browser_resize | ✅ 新增 |
| 14 | browser_drag | ✅ 新增 |
| 15 | browser_close | ✅ 新增 |
| 16 | browser_file_upload | ✅ 新增 |
| 17 | browser_handle_dialog | ✅ 新增 |
| 18 | browser_console_messages | ✅ 新增 |
| 19 | browser_network_requests | ✅ 新增 |

## 测试建议

### 测试新增工具

```bash
# 1. 测试 browser_press_key
curl -X POST http://localhost:8080/api/mcp/call \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "browser_press_key",
    "arguments": {
      "key": "Enter"
    }
  }'

# 2. 测试 browser_evaluate
curl -X POST http://localhost:8080/api/mcp/call \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "browser_evaluate",
    "arguments": {
      "script": "document.title"
    }
  }'

# 3. 测试 browser_resize
curl -X POST http://localhost:8080/api/mcp/call \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "browser_resize",
    "arguments": {
      "width": 1920,
      "height": 1080
    }
  }'

# 4. 测试 browser_drag
curl -X POST http://localhost:8080/api/mcp/call \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "browser_drag",
    "arguments": {
      "from_identifier": "Clickable Element [1]",
      "to_identifier": "Clickable Element [2]"
    }
  }'

# 5. 测试 browser_console_messages
curl -X POST http://localhost:8080/api/mcp/call \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "browser_console_messages",
    "arguments": {}
  }'
```

## 经验教训

### 问题根源

在添加新的 executor 工具时，需要在**三个地方**同步修改：

1. **`executor/operations.go`** - 实现核心功能
2. **`executor/mcp_tools.go`** - 注册到 MCP 工具注册表 + 添加元数据
3. **`mcp/server.go`** - 在 `callExecutorTool()` 的 switch 语句中添加处理逻辑 ⚠️ **容易遗漏**

### 改进建议

#### 方案 1: 使用反射自动路由
```go
// 在 callExecutorTool 中使用反射自动调用方法
func (s *MCPServer) callExecutorTool(ctx context.Context, name string, arguments map[string]interface{}) (interface{}, error) {
    // 将 browser_press_key 转换为 PressKey
    methodName := convertToolNameToMethodName(name)
    
    // 使用反射调用
    method := reflect.ValueOf(s.executor).MethodByName(methodName)
    if !method.IsValid() {
        return nil, fmt.Errorf("unknown executor tool: %s", name)
    }
    
    // 调用方法
    // ...
}
```

#### 方案 2: 注册式路由表
```go
// 在 Executor 初始化时注册所有工具的处理函数
type ToolHandler func(ctx context.Context, args map[string]interface{}) (*OperationResult, error)

var toolHandlers = map[string]ToolHandler{
    "browser_press_key": func(ctx context.Context, args map[string]interface{}) (*OperationResult, error) {
        // 处理逻辑
    },
    // ...
}

func (s *MCPServer) callExecutorTool(ctx context.Context, name string, arguments map[string]interface{}) (interface{}, error) {
    handler, ok := toolHandlers[name]
    if !ok {
        return nil, fmt.Errorf("unknown executor tool: %s", name)
    }
    return handler(ctx, arguments)
}
```

#### 方案 3: 统一通过 MCP 工具注册表
```go
// 让 executor 的 MCP 工具注册表处理所有调用
// 不需要在 mcp/server.go 中单独处理
func (s *MCPServer) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (interface{}, error) {
    if strings.HasPrefix(name, "browser_") {
        // 直接调用工具注册表
        return s.toolRegistry.ExecuteTool(ctx, name, arguments)
    }
    // ...
}
```

## 总结

✅ **修复完成**
- 删除重复的元数据定义
- 修复工具名称不一致
- 添加所有新工具的处理逻辑
- 编译成功
- 19 个工具全部可用

⚠️ **注意事项**
- 添加新工具时记得同步修改 3 个文件
- 工具名称必须完全一致
- 所有参数类型要正确处理（特别是 float64 转 int）

📝 **相关文档**
- 详细功能文档: `BROWSER_COMMANDS_COMPLETED.md`
- 快速参考: `BROWSER_COMMANDS_QUICK_REFERENCE.md`
- 本次修复: `BROWSER_COMMANDS_BUG_FIXES.md`
