# 脚本预设变量功能

## 功能概述

为脚本系统添加了预设变量功能，允许在脚本中定义可复用的变量，这些变量可以在条件判断、分支和迭代中使用。变量使用 `${变量名}` 格式引用，支持外部调用时传入参数覆盖默认值。

## 功能特性

1. **变量定义**：在脚本中定义预设变量及其默认值
2. **变量引用**：在脚本的各个字段中使用 `${变量名}` 引用变量
3. **参数覆盖**：外部调用时可传入参数覆盖预设变量的默认值
4. **导入导出**：脚本导入导出时自动包含变量定义
5. **可视化管理**：前端提供友好的变量管理界面

## 技术实现

### 后端修改

#### 1. 数据模型 (`backend/models/script.go`)

在 `Script` 结构体中添加了 `Variables` 字段：

```go
// 预设变量（可以在脚本中使用 ${变量名} 引用，也可以在外部调用时传入覆盖）
Variables map[string]string `json:"variables,omitempty"` // 预设变量，key 为变量名，value 为默认值
```

同时更新了 `Copy()` 方法以正确复制变量。

#### 2. MCP 服务器 (`backend/mcp/server.go`)

在 `createToolHandler` 函数中实现了变量合并逻辑：

```go
// 合并参数：先使用脚本预设变量，再用外部传入的参数覆盖
params := make(map[string]string)

// 1. 首先添加脚本的预设变量
if scriptToRun.Variables != nil {
    for key, value := range scriptToRun.Variables {
        params[key] = value
    }
}

// 2. 外部传入的参数会覆盖预设变量
if request.Params.Arguments != nil {
    if argsMap, ok := request.Params.Arguments.(map[string]interface{}); ok {
        for key, value := range argsMap {
            params[key] = fmt.Sprintf("%v", value)
        }
    }
}
```

#### 3. API 处理器 (`backend/api/handlers.go`)

在 `PlayScript` 函数中实现了相同的变量合并逻辑，确保 HTTP API 调用也支持预设变量和参数覆盖。

### 前端修改

#### 1. 类型定义 (`frontend/src/api/client.ts`)

在 `Script` 和 `SaveScriptRequest` 接口中添加了 `variables` 字段：

```typescript
export interface Script {
  // ... 其他字段
  variables?: Record<string, string>  // 预设变量
}

export interface SaveScriptRequest {
  // ... 其他字段
  variables?: Record<string, string>  // 预设变量
}
```

#### 2. 脚本管理器 (`frontend/src/pages/ScriptManager.tsx`)

添加了完整的变量管理功能：

**状态管理**：
- `editingVariables`: 当前编辑的变量
- `newVariableName`: 新变量名输入
- `newVariableValue`: 新变量值输入

**变量操作函数**：
- `handleAddVariable()`: 添加新变量
- `handleUpdateVariable()`: 更新变量值
- `handleDeleteVariable()`: 删除变量

**UI 界面**：

编辑模式下显示变量管理区域：
- 显示所有已定义的变量，支持修改和删除
- 提供输入框添加新变量
- 使用紫色主题突出显示变量

非编辑模式下显示变量摘要：
- 以紫色标签形式显示所有变量名

**导入导出**：
- `performImport()` 函数在创建和更新脚本时都会包含 variables 字段
- 导出功能自动包含 variables（通过 rest operator）

## 使用示例

### 1. 在脚本中定义变量

在前端脚本编辑器中：
1. 点击编辑脚本
2. 在"脚本变量"区域添加变量：
   - 变量名：`username`
   - 默认值：`testuser`
3. 在脚本的 action 中使用：`${username}`

### 2. 外部调用时覆盖变量

**通过 HTTP API**：
```bash
curl -X POST http://localhost:8080/api/v1/scripts/{id}/play \
  -H "Content-Type: application/json" \
  -d '{
    "params": {
      "username": "realuser"
    }
  }'
```

**通过 MCP 命令**：
```json
{
  "method": "tools/call",
  "params": {
    "name": "execute_login_script",
    "arguments": {
      "username": "realuser"
    }
  }
}
```

### 3. 变量的优先级

参数合并遵循以下优先级（从低到高）：
1. 脚本预设变量的默认值
2. 外部调用传入的参数

这意味着外部传入的参数会覆盖预设变量的默认值。

## 应用场景

1. **通用脚本模板**：定义可复用的脚本，通过变量适配不同场景
2. **测试与生产环境**：使用不同的变量值区分测试和生产环境
3. **批量操作**：通过循环调用脚本并传入不同的变量值实现批量操作
4. **条件逻辑**：（未来）可基于变量值实现条件分支和循环

## 后续扩展

1. **条件和分支**：基于变量值实现 if-else 逻辑
2. **循环迭代**：基于变量实现循环操作
3. **变量类型**：支持更多变量类型（数字、布尔值、数组等）
4. **变量表达式**：支持变量间的计算和引用
5. **SDK 支持**：在 SDK 的 Play 函数中添加参数支持

## 兼容性

- 向后兼容：现有没有变量的脚本不受影响
- 变量字段为可选（`omitempty`），不会影响现有数据结构
- 导入导出兼容新旧格式

## 测试建议

1. **基础功能测试**：
   - 创建带变量的脚本
   - 编辑变量值
   - 删除变量
   - 保存并重新加载脚本

2. **参数覆盖测试**：
   - 使用预设值执行脚本
   - 传入参数覆盖预设值执行脚本
   - 验证参数优先级

3. **导入导出测试**：
   - 导出带变量的脚本
   - 导入脚本验证变量完整性

4. **集成测试**：
   - MCP 命令调用带变量的脚本
   - HTTP API 调用带变量的脚本
