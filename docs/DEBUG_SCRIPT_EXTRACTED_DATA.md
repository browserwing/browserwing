# 调试脚本工具 ExtractedData 返回问题

## ✅ 问题已解决

**根本原因**: MCP Server 有两个返回路径，只修复了一个。

### 问题分析

从用户提供的日志发现：
```
[PlayScript] Extracted data keys: [ai_data_3]  ✅ 脚本抓取到了数据
[Agent MCPTool] Result map keys: [extracted_data success message]  ⚠️ extracted_data 在顶层
[Agent MCPTool] No data field found in result  ❌ Agent 找不到 data 字段
```

MCP Server 有两个调用路径：
1. **createToolHandler** (第 217-304 行) - MCP 协议调用 → 已修复 ✅
2. **CallTool** (第 405-487 行) - Agent 直接调用 → **这里有问题** ❌

### 修复方案

修改 `CallTool` 方法的返回结构，将 `extracted_data` 放在 `data` 字段中：

```go
// 修改前
result := map[string]interface{}{
    "success": playResult.Success,
    "message": playResult.Message,
}
if len(playResult.ExtractedData) > 0 {
    result["extracted_data"] = playResult.ExtractedData  // ❌ 顶层
}

// 修改后
result := map[string]interface{}{
    "success": playResult.Success,
    "message": playResult.Message,
}
if len(playResult.ExtractedData) > 0 {
    result["data"] = map[string]interface{}{
        "extracted_data": playResult.ExtractedData,  // ✅ 嵌套在 data 中
    }
}
```

---

## 原问题描述

用户反馈：Agent 调用脚本工具时只返回了 message（"Script replay completed"），但没有返回抓取的数据（ExtractedData）。

## 添加的调试日志

为了排查这个问题，我在数据流的关键节点添加了详细的日志：

### 1. PlayScript 返回阶段 (`backend/services/browser/manager.go`)

```go
extractedData := player.GetExtractedData()
logger.Info(ctx, "[PlayScript] Extracted data length: %d", len(extractedData))
if len(extractedData) > 0 {
    keys := make([]string, 0, len(extractedData))
    for k := range extractedData {
        keys = append(keys, k)
    }
    logger.Info(ctx, "[PlayScript] Extracted data keys: %v", keys)
}
```

**作用**: 检查脚本执行后是否真的抓取到了数据

### 2. MCP Server 返回阶段 (`backend/mcp/server.go`)

```go
logger.Info(ctx, "[MCP Script Tool] ExtractedData length: %d", len(playResult.ExtractedData))
if len(playResult.ExtractedData) > 0 {
    logger.Info(ctx, "[MCP Script Tool] ExtractedData keys: %v", getKeysFromMap(playResult.ExtractedData))
    logger.Info(ctx, "[MCP Script Tool] Added extracted_data to result")
} else {
    logger.Info(ctx, "[MCP Script Tool] No extracted data to return")
}
```

**作用**: 检查 MCP server 是否正确接收到数据并构建返回结果

### 3. Agent 处理阶段 (`backend/agent/agent.go`)

```go
logger.Info(ctx, "[Agent MCPTool] Result map keys: %v", getMapKeys(resultMap))

if message, ok := resultMap["message"].(string); ok {
    logger.Info(ctx, "[Agent MCPTool] Got message: %s", message)
}

if data, ok := resultMap["data"].(map[string]interface{}); ok {
    logger.Info(ctx, "[Agent MCPTool] Found data field with keys: %v", getMapKeys(data))
    // ... 处理数据
    logger.Info(ctx, "[Agent MCPTool] Added data to response for tool: %s (data keys: %v)", t.name, getMapKeys(data))
} else {
    logger.Info(ctx, "[Agent MCPTool] No data field found in result")
}
```

**作用**: 检查 Agent 是否正确接收和处理 data 字段

## 如何使用这些日志排查问题

### 步骤 1: 准备测试脚本

创建一个包含数据抓取操作的测试脚本：

```json
{
  "name": "Test Extract",
  "url": "https://example.com",
  "actions": [
    {
      "type": "extract",
      "selector": "h1",
      "attribute": "text",
      "variable_name": "title"
    },
    {
      "type": "extract",
      "selector": ".price",
      "attribute": "text",
      "variable_name": "price"
    }
  ],
  "is_mcp_command": true,
  "mcp_command_name": "test_extract"
}
```

### 步骤 2: 启动后端并查看日志

```bash
cd /root/code/browserwing/backend
./browserwing 2>&1 | tee debug.log
```

### 步骤 3: 通过 Agent 调用脚本工具

在 Agent Chat 中发送消息：
```
请执行 test_extract 脚本
```

### 步骤 4: 分析日志输出

根据日志输出来判断问题出在哪个阶段：

#### 场景 1: 脚本没有抓取到数据

**日志输出**:
```
[PlayScript] Extracted data length: 0
[MCP Script Tool] ExtractedData length: 0
[MCP Script Tool] No extracted data to return
```

**问题**: 脚本执行了，但没有抓取到数据
**可能原因**:
1. 选择器不正确（元素不存在）
2. 页面还没加载完就执行了抓取
3. 脚本中没有 extract 类型的 action
4. extract action 的 variable_name 为空

**解决方法**:
- 检查脚本中的选择器是否正确
- 在 extract 前添加 wait 操作
- 确保 variable_name 已设置

#### 场景 2: 抓取到了数据，但 MCP 没有返回

**日志输出**:
```
[PlayScript] Extracted data length: 2
[PlayScript] Extracted data keys: [title price]
[MCP Script Tool] ExtractedData length: 0  ❌ 不一致
```

**问题**: PlayScript 返回了数据，但 MCP server 没有接收到
**可能原因**:
- PlayResult 结构传递问题
- 数据在中间被清空

**解决方法**:
- 检查 PlayResult 的创建和传递
- 检查是否有地方修改了 ExtractedData

#### 场景 3: MCP 返回了数据，但 Agent 没有处理

**日志输出**:
```
[PlayScript] Extracted data length: 2
[PlayScript] Extracted data keys: [title price]
[MCP Script Tool] ExtractedData length: 2
[MCP Script Tool] ExtractedData keys: [title price]
[MCP Script Tool] Added extracted_data to result
[Agent MCPTool] Result map keys: [success message]  ❌ 没有 data
[Agent MCPTool] No data field found in result
```

**问题**: MCP server 构建了数据，但 Agent 没有收到 data 字段
**可能原因**:
- `mcpgo.NewToolResultJSON` 的序列化问题
- 数据结构不正确

**解决方法**:
- 检查 `resultData` 的构建
- 打印完整的 `resultData` 看结构

#### 场景 4: Agent 收到了数据，但没有添加到响应

**日志输出**:
```
[PlayScript] Extracted data length: 2
[PlayScript] Extracted data keys: [title price]
[MCP Script Tool] ExtractedData length: 2
[MCP Script Tool] Added extracted_data to result
[Agent MCPTool] Result map keys: [success message data]  ✅
[Agent MCPTool] Found data field with keys: [extracted_data]  ✅
[Agent MCPTool] Added data to response for tool: xxx (data keys: [extracted_data])  ✅
```

**问题**: 所有日志都正常，但前端还是看不到数据
**可能原因**:
- 前端显示问题
- 响应文本被截断
- SSE 流式传输问题

**解决方法**:
- 检查前端的工具结果显示逻辑
- 检查 SSE 流式传输是否完整

## 正常情况的完整日志示例

```
2026-01-17 10:30:15 [INFO] Executing MCP command: test_extract (script: Test Extract)
2026-01-17 10:30:15 [INFO] Browser not running, starting...
2026-01-17 10:30:18 [INFO] Browser started successfully
2026-01-17 10:30:20 [INFO] [PlayScript] Extracted data length: 2
2026-01-17 10:30:20 [INFO] [PlayScript] Extracted data keys: [title price]
2026-01-17 10:30:20 [INFO] [MCP Script Tool] ExtractedData length: 2
2026-01-17 10:30:20 [INFO] [MCP Script Tool] ExtractedData keys: [title price]
2026-01-17 10:30:20 [INFO] [MCP Script Tool] Added extracted_data to result
2026-01-17 10:30:20 [INFO] [Agent MCPTool] Result map keys: [success message data]
2026-01-17 10:30:20 [INFO] [Agent MCPTool] Got message: Script replay completed
2026-01-17 10:30:20 [INFO] [Agent MCPTool] Found data field with keys: [extracted_data]
2026-01-17 10:30:20 [INFO] [Agent MCPTool] Added data to response for tool: test_extract (data keys: [extracted_data])
```

## 常见问题排查

### Q1: 日志显示 "Extracted data length: 0"

**检查项**:
1. 脚本中是否有 extract 类型的 action？
   ```json
   {
     "type": "extract",
     "selector": "h1",
     "attribute": "text",
     "variable_name": "title"  // ⚠️ 必须设置
   }
   ```

2. variable_name 是否设置？
   - 如果没有设置 variable_name，数据不会被保存

3. 选择器是否能找到元素？
   - 在浏览器控制台测试：`document.querySelector("h1")`
   - 如果返回 null，说明选择器不对

4. 页面是否加载完成？
   - 在 extract 前添加 wait 操作：
   ```json
   {
     "type": "wait",
     "selector": "h1",
     "timeout": 5000
   }
   ```

### Q2: PlayScript 有数据，但 MCP 收到的是空

**调试方法**:
在 `backend/mcp/server.go` 的 `createToolHandler` 中添加打印：
```go
logger.Info(ctx, "[DEBUG] playResult: %+v", playResult)
```

查看 playResult 的完整内容。

### Q3: MCP 返回了 data，但 Agent 没有收到

**调试方法**:
在 `backend/mcp/server.go` 返回前添加打印：
```go
resultJSON, _ := json.MarshalIndent(resultData, "", "  ")
logger.Info(ctx, "[DEBUG] resultData: %s", string(resultJSON))
```

查看实际返回的 JSON 结构。

### Q4: Agent 处理了数据，但前端看不到

**检查前端**:
1. 打开浏览器控制台
2. 查看 Network 标签，找到 SSE 连接
3. 查看响应内容是否包含数据

**检查后端响应**:
在 `backend/agent/agent.go` 的 `Execute` 返回前打印：
```go
logger.Info(ctx, "[DEBUG] Final responseText length: %d", len(responseText))
logger.Info(ctx, "[DEBUG] First 500 chars: %s", responseText[:min(500, len(responseText))])
```

## 下一步

1. **重启后端**:
   ```bash
   cd /root/code/browserwing/backend
   pkill -f browserwing
   ./browserwing 2>&1 | tee debug.log
   ```

2. **测试脚本工具**:
   - 创建或使用一个包含 extract 操作的脚本
   - 确保 variable_name 已设置
   - 通过 Agent 调用

3. **查看日志**:
   ```bash
   # 实时查看日志
   tail -f debug.log | grep -E "\[PlayScript\]|\[MCP Script Tool\]|\[Agent MCPTool\]"
   
   # 或者只看错误
   tail -f debug.log | grep -i error
   ```

4. **根据日志输出判断问题所在**:
   - 参考上面的场景分析
   - 找到数据丢失的环节

5. **反馈日志**:
   - 将相关日志发给我
   - 我会帮你分析具体问题

## 测试用例

### 简单测试脚本

```json
{
  "name": "Simple Extract Test",
  "url": "https://example.com",
  "actions": [
    {
      "type": "extract",
      "selector": "h1",
      "attribute": "text",
      "variable_name": "page_title"
    }
  ],
  "is_mcp_command": true,
  "mcp_command_name": "simple_test"
}
```

**期望的日志输出**:
```
[PlayScript] Extracted data length: 1
[PlayScript] Extracted data keys: [page_title]
[MCP Script Tool] ExtractedData length: 1
[MCP Script Tool] ExtractedData keys: [page_title]
[MCP Script Tool] Added extracted_data to result
[Agent MCPTool] Result map keys: [success message data]
[Agent MCPTool] Found data field with keys: [extracted_data]
[Agent MCPTool] Added data to response
```

**期望的前端显示**:
```
Tool: simple_test
Status: Success
Result:
Script replay completed

Data:
{
  "extracted_data": {
    "page_title": "Example Domain"
  }
}
```

## 总结

通过这些调试日志，我们可以精确定位问题出在数据流的哪个环节：

1. **PlayScript** → 脚本是否抓取到数据
2. **MCP Server** → 数据是否正确构建和返回
3. **Agent** → 数据是否正确接收和处理
4. **Frontend** → 数据是否正确显示

请按照上面的步骤测试，并将日志输出发给我，我会帮你分析具体问题。
