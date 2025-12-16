# MCP 命令集成使用说明

## 功能概述

BrowserPilot 现在支持将录制的自动化脚本转换为 MCP (Model Context Protocol) 命令,可以被外部 MCP 客户端调用。

**支持两种通信模式:**
- **HTTP/SSE 模式** (推荐): 通过 HTTP API 调用,支持远程访问
- **stdio 模式**: 通过标准输入/输出通信,适合本地进程

## 主要特性

- ✅ 将任何脚本设置为 MCP 命令
- ✅ 自定义命令名称和描述
- ✅ 动态注册/取消注册命令
- ✅ 支持 MCP 标准协议 (2024-11-05)
- ✅ 自动执行脚本并返回抓取数据
- ✅ stdio 模式通信

## 如何使用

### 1. 在前端设置 MCP 命令

1. 进入"脚本管理"页面
2. 找到要设置为 MCP 命令的脚本
3. 点击脚本操作栏的 MCP 按钮(紫色图标)
4. 在弹出的对话框中配置:
   - **MCP 命令名称**: 例如 `execute_login_script` (必填,使用小写字母和下划线)
   - **MCP 命令描述**: 描述命令的功能和用途
   - **输入参数定义** (可选): 使用 JSON Schema 格式定义命令接受的参数
5. 点击"保存"

#### 参数定义示例

如果你的脚本需要接收外部参数,可以定义 Input Schema:

```json
{
  "type": "object",
  "properties": {
    "username": {
      "type": "string",
      "description": "登录用户名"
    },
    "password": {
      "type": "string",
      "description": "登录密码"
    },
    "remember": {
      "type": "boolean",
      "description": "是否记住登录",
      "default": false
    }
  },
  "required": ["username", "password"]
}
```

**参数使用说明**: 
- 在脚本中使用 `${参数名}` 格式的占位符,例如 `${username}`, `${password}`
- 占位符可以用在:
  - 脚本 URL
  - Action 的 Selector、XPath、Value、URL、JSCode
  - 文件路径 (FilePaths)
- 执行时,MCP 服务器会自动将占位符替换为实际参数值
- 如果参数未提供,占位符会被替换为空字符串

**示例:**
```
录制脚本时在输入框的 Value 中使用: ${username}
调用 MCP 命令时传入: {"username": "test@example.com"}
实际执行时会替换为: test@example.com
```

### 2. MCP 服务器会自动启动

MCP 服务器在应用启动时自动初始化,并加载所有已设置为 MCP 命令的脚本。

### 3. 通过 MCP 客户端调用

MCP 服务器支持标准的 MCP 协议,提供两种调用方式:

#### 方式 1: HTTP/SSE 模式 (推荐)

**优点:**
- 可以远程访问
- 多客户端共享同一服务器实例
- 支持 REST API 直接调用
- 适合微服务和云环境

**配置示例 (Cline/Continue):**
```json
{
  "mcpServers": {
    "browserpilot": {
      "url": "http://localhost:8080/api/v1/mcp/message"
    }
  }
}
```

**直接 HTTP 调用:**
```bash
curl -X POST http://localhost:8080/api/v1/mcp/message \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list"
  }'
```

#### 方式 2: stdio 模式

**优点:**
- 简单安全,不需要网络端口
- 进程隔离,资源管理清晰
- 符合 MCP 标准推荐方式

**配置示例:**

**配置示例:**
```bash
# stdio 模式 - 启动独立进程
./browserwing
```

**Cline/Continue 配置:**
```json
{
  "mcpServers": {
    "browserpilot": {
      "command": "/path/to/browserwing",
      "args": []
    }
  }
}
```

#### MCP 协议交互示例

**初始化请求:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize"
}
```

**列出所有工具:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list"
}
```

**调用工具 (执行脚本):**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "execute_login_script",
    "arguments": {
      "username": "user@example.com",
      "password": "secret123"
    }
  }
}
```

**响应示例:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"success\": true, \"message\": \"脚本执行成功\", \"extracted_data\": {...}}"
      }
    ]
  }
}
```

### 4. 查看 MCP 服务状态

通过 API 查看当前注册的 MCP 命令:

```bash
# 获取 MCP 服务状态
curl http://localhost:8080/api/v1/mcp/status

# 列出所有 MCP 命令
curl http://localhost:8080/api/v1/mcp/commands

# 测试 HTTP 模式的 MCP 调用
curl -X POST http://localhost:8080/api/v1/mcp/message \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list"
  }'
```

## MCP 端点说明

### HTTP/SSE 模式端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/mcp/message` | POST | 处理 MCP JSON-RPC 请求 |
| `/api/v1/mcp/sse` | GET | SSE 长连接（用于事件推送） |
| `/api/v1/mcp/status` | GET | 获取 MCP 服务状态 |
| `/api/v1/mcp/commands` | GET | 列出所有 MCP 命令 |

### 使用示例

**列出工具:**
```bash
curl -X POST http://localhost:8080/api/v1/mcp/message \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/list"
  }'
```

**调用工具:**
```bash
curl -X POST http://localhost:8080/api/v1/mcp/message \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "execute_login_script",
      "arguments": {
        "username": "user@example.com",
        "password": "secret123"
      }
    }
  }'
```

## API 接口

### 设置/取消 MCP 命令

```http
POST /api/v1/scripts/:id/mcp
Content-Type: application/json

{
  "is_mcp_command": true,
  "mcp_command_name": "execute_login_script",
  "mcp_command_description": "执行登录脚本",
  "mcp_input_schema": {
    "type": "object",
    "properties": {
      "username": {
        "type": "string",
        "description": "用户名"
      }
    }
  }
}
```

### 获取 MCP 服务状态

```http
GET /api/v1/mcp/status
```

响应:
```json
{
  "running": true,
  "commands": [
    {
      "name": "execute_login_script",
      "description": "执行登录脚本",
      "script_name": "自动登录脚本",
      "script_id": "abc123"
    }
  ],
  "command_count": 1
}
```

### 列出所有 MCP 命令

```http
GET /api/v1/mcp/commands
```

## 实现细节

### 后端架构

```
backend/
├── mcp/
│   └── server.go          # MCP 服务器实现
├── models/
│   └── script.go          # Script 模型添加 MCP 字段
├── api/
│   ├── handlers.go        # MCP API 处理器
│   └── router.go          # MCP 路由
└── main.go                # MCP 服务器启动和集成
```

### 前端界面

- 脚本列表中每个脚本显示 MCP 状态(紫色图标)
- 点击图标可以设置/取消 MCP 命令
- 配置对话框用于设置命令名称和描述

### MCP 协议支持

- ✅ initialize
- ✅ tools/list
- ✅ tools/call
- ✅ 标准错误处理
- ✅ JSON-RPC 2.0 格式

## 使用场景

1. **AI 助手集成**: 将脚本作为 AI 助手的工具,让 AI 可以自动执行浏览器操作
2. **工作流自动化**: 在工作流中通过 HTTP API 调用浏览器自动化脚本
3. **批量操作**: 通过 MCP 客户端批量执行多个脚本
4. **跨系统集成**: 让其他系统能够调用浏览器自动化能力
5. **远程调用**: HTTP 模式支持跨机器远程调用自动化脚本

## 模式对比

| 特性 | HTTP/SSE 模式 | stdio 模式 |
|------|--------------|-----------|
| 远程访问 | ✅ 支持 | ❌ 仅本地 |
| 多客户端共享 | ✅ 支持 | ❌ 独立进程 |
| REST API | ✅ 支持 | ❌ 不支持 |
| 安全性 | 需要网络安全 | 本地进程隔离 |
| 部署方式 | 中心化服务 | 分布式进程 |
| 适用场景 | 微服务/云环境 | 桌面应用 |

## 注意事项

1. **命令名称唯一性**: 每个 MCP 命令名称必须唯一
2. **命名规范**: 建议使用小写字母和下划线,例如 `execute_login_script`
3. **浏览器自动启动**: MCP 命令执行时，如果浏览器未运行，会自动启动浏览器
4. **自动清理页面**: 脚本执行完成后，会自动关闭创建的页面，避免页面堆积
5. **参数占位符**: 在脚本中使用 `${参数名}` 格式定义占位符，执行时会自动替换为实际参数值
6. **占位符位置**: 占位符可用于 URL、Selector、XPath、Value、JSCode、FilePaths 等字段
7. **并发执行**: 多个命令可以并发执行,每个命令使用独立的浏览器会话
8. **数据抓取**: 如果脚本包含数据抓取步骤,结果会在响应中的 `extracted_data` 字段返回

## 示例配置

### Claude Desktop / Cline 集成 (HTTP 模式)

在配置文件中添加:

```json
{
  "mcpServers": {
    "browserpilot": {
      "url": "http://localhost:8080/api/v1/mcp/message",
      "transport": "http"
    }
  }
}
```

### Claude Desktop 集成 (stdio 模式)

在 `claude_desktop_config.json` 中添加:

```json
{
  "mcpServers": {
    "browserpilot": {
      "command": "/path/to/browserwing",
      "args": []
    }
  }
}
```

### Cursor 集成 (HTTP 模式)

在 Cursor 设置中添加 MCP 服务器:

```json
{
  "mcp.servers": {
    "browserpilot": {
      "url": "http://localhost:8080/api/v1/mcp/message"
    }
  }
}
```

### Cursor 集成 (stdio 模式)

```json
{
  "mcp.servers": {
    "browserpilot": {
      "command": "/path/to/browserwing"
    }
  }
}
```

### Continue 扩展集成 (HTTP 模式)

在 `~/.continue/config.json` 中:

```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "name": "browserpilot",
        "url": "http://localhost:8080/api/v1/mcp/message"
      }
    ]
  }
}
```

## 故障排查

### MCP 服务器未启动

检查日志中是否有 "✓ MCP 服务器初始化成功" 消息。

### 命令未注册

1. 确认脚本已设置为 MCP 命令
2. 检查命令名称是否唯一
3. 通过 `/api/v1/mcp/status` 查看当前注册的命令

### 命令执行失败

1. 检查脚本本身是否可以正常回放
2. 查看日志中的错误信息
3. 确认浏览器实例正常运行

## 后续计划

- [ ] 支持命令参数验证
- [ ] 支持命令执行超时设置
- [ ] 支持命令执行结果缓存
- [ ] 支持命令执行历史记录
- [ ] 提供 WebSocket 模式通信
- [ ] 提供 REST API 模式调用

## 技术支持

如有问题,请查看:
- 应用日志: `logs/browserwing.log`
- 浏览器日志: 开发者工具控制台
- API 文档: 查看项目 README
