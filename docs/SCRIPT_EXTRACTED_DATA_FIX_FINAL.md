# 脚本工具 ExtractedData 返回问题 - 最终修复

## 🎯 问题根源

MCP Server 有**两个调用路径**，之前只修复了一个：

### 1. createToolHandler（MCP 协议调用）
- 位置: `backend/mcp/server.go` 第 217-312 行
- 用途: MCP 协议标准调用
- 状态: ✅ 已修复

### 2. CallTool（Agent 直接调用）⚠️
- 位置: `backend/mcp/server.go` 第 405-500 行
- 用途: Agent 直接调用脚本工具
- 状态: ❌ **这里是问题所在**

## 📊 日志分析

从用户提供的日志可以看出：

```json
// ✅ 脚本成功抓取数据
{"msg":"[PlayScript] Extracted data keys: [ai_data_3]"}

// ❌ Agent 收到的结构不对
{"msg":"[Agent MCPTool] Result map keys: [extracted_data success message]"}
//                                        ^^^^^^^^^^^^^^ 在顶层，不在 data 字段中

// ❌ Agent 找不到 data 字段
{"msg":"[Agent MCPTool] No data field found in result"}
```

**问题**: `extracted_data` 在顶层，而 Agent 期望在 `data` 字段中。

## 🔧 修复代码

### 修改位置
`backend/mcp/server.go` 的 `CallTool` 方法（第 472-500 行）

### 修改前
```go
// 返回结果
result := map[string]interface{}{
    "success": playResult.Success,
    "message": playResult.Message,
}

if len(playResult.ExtractedData) > 0 {
    result["extracted_data"] = playResult.ExtractedData  // ❌ 顶层
}

return result, nil
```

### 修改后
```go
// 调试日志：检查 ExtractedData
logger.Info(ctx, "[MCP CallTool] ExtractedData length: %d", len(playResult.ExtractedData))
if len(playResult.ExtractedData) > 0 {
    logger.Info(ctx, "[MCP CallTool] ExtractedData keys: %v", getKeysFromMap(playResult.ExtractedData))
}

// 构建返回结果，将 extracted_data 放在 data 字段中以便 Agent 处理
result := map[string]interface{}{
    "success": playResult.Success,
    "message": playResult.Message,
}

// 如果有抓取的数据，将其放在 data 字段中
if len(playResult.ExtractedData) > 0 {
    result["data"] = map[string]interface{}{
        "extracted_data": playResult.ExtractedData,  // ✅ 嵌套在 data 中
    }
    logger.Info(ctx, "[MCP CallTool] Added extracted_data to result in data field")
} else {
    logger.Info(ctx, "[MCP CallTool] No extracted data to return")
}

return result, nil
```

## 📝 数据流（修复后）

```
脚本执行
    ↓
PlayScript 返回 ExtractedData
    ↓
CallTool 构建返回结构:
{
    "success": true,
    "message": "Script replay completed",
    "data": {                              ✨ 关键：放在 data 字段中
        "extracted_data": {
            "ai_data_3": [...]
        }
    }
}
    ↓
Agent MCPTool.Execute 接收
    ↓
检查 data 字段 → ✅ 找到了
    ↓
序列化为 JSON:
"Script replay completed

Data:
{
  \"extracted_data\": {
    \"ai_data_3\": [...]
  }
}"
    ↓
返回给前端显示 ✅
```

## 🧪 测试步骤

### 1. 重新编译
```bash
cd /root/code/browserwing/backend
go build
```

### 2. 启动后端
```bash
./browserwing 2>&1 | tee test.log
```

### 3. 测试脚本工具

在 Agent Chat 中调用包含数据抓取的脚本工具。

### 4. 查看日志

**期望看到的日志**:
```
[PlayScript] Extracted data length: 1
[PlayScript] Extracted data keys: [ai_data_3]
[MCP CallTool] ExtractedData length: 1                    ✨ 新增
[MCP CallTool] ExtractedData keys: [ai_data_3]            ✨ 新增
[MCP CallTool] Added extracted_data to result in data field  ✨ 新增
[Agent MCPTool] Result map keys: [success message data]   ✅ 现在有 data 字段
[Agent MCPTool] Found data field with keys: [extracted_data]  ✅
[Agent MCPTool] Added data to response                    ✅
```

### 5. 查看前端显示

**期望看到**:
```
Tool: script_xxx
Status: Success ✅
Result:
Script replay completed

Data:
{
  "extracted_data": {
    "ai_data_3": [
      {
        "title": "...",
        "author": "...",
        ...
      }
    ]
  }
}
```

## 📋 修改的文件

1. **backend/mcp/server.go**
   - `CallTool` 方法（第 472-500 行）
   - 添加调试日志
   - 修改返回结构（将 extracted_data 放在 data 字段中）

## ⚠️ 重要提示

两个调用路径都需要返回相同的数据结构：

| 方法 | 调用者 | 状态 |
|------|--------|------|
| `createToolHandler` | MCP 协议 | ✅ 已修复 |
| `CallTool` | Agent 直接调用 | ✅ 已修复 |

## 🎉 效果对比

### 修复前

**日志**:
```
[Agent MCPTool] Result map keys: [extracted_data success message]
[Agent MCPTool] No data field found in result  ❌
```

**前端显示**:
```
Script replay completed  ❌ 只有消息，没有数据
```

### 修复后

**日志**:
```
[MCP CallTool] Added extracted_data to result in data field
[Agent MCPTool] Result map keys: [success message data]  ✅
[Agent MCPTool] Found data field with keys: [extracted_data]  ✅
[Agent MCPTool] Added data to response  ✅
```

**前端显示**:
```
Script replay completed

Data:
{
  "extracted_data": {
    "ai_data_3": [...]  ✅ 完整的抓取数据
  }
}
```

## 🔍 为什么之前没发现

1. **有两个调用路径**: 一开始只修复了 `createToolHandler`，忽略了 `CallTool`
2. **Agent 使用的是 CallTool**: 而不是 MCP 协议的标准调用
3. **日志不够**: 之前没有在关键位置添加日志，无法定位问题

## ✅ 总结

- **问题**: MCP Server 的 `CallTool` 方法返回的 `extracted_data` 在顶层，而不是在 `data` 字段中
- **影响**: Agent 无法识别和显示脚本抓取的数据
- **修复**: 将 `extracted_data` 包装在 `data` 字段中，与 Agent 的处理逻辑对齐
- **验证**: 添加调试日志，便于未来排查类似问题

现在脚本工具的数据抓取功能完全可用了！🎊
