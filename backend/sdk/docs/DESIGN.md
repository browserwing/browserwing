# BrowserWing SDK 设计与实现

## 概述

BrowserWing SDK 是一个 Go 语言的软件开发工具包,允许其他 Go 项目以编程方式集成和使用 BrowserWing 的核心功能。SDK 采用模块化设计,支持按需启用功能模块。

## 设计目标

1. **模块化**: 功能模块可独立启用/禁用,避免不必要的依赖
2. **易用性**: 提供简洁清晰的 API,降低学习成本
3. **类型安全**: 充分利用 Go 的类型系统,减少运行时错误
4. **向后兼容**: 完全基于现有项目结构,无需修改核心代码

## 架构设计

### 核心组件

```
sdk/
├── client.go       # 主客户端,管理所有模块
├── browser.go      # 浏览器管理模块
├── script.go       # 脚本管理模块
├── agent.go        # Agent 对话模块
├── types.go        # 类型定义
├── README.md       # SDK 介绍
├── USAGE.md        # 使用文档
└── examples/       # 示例代码
    ├── basic/      # 基础示例
    ├── agent/      # Agent 示例
    └── advanced/   # 高级示例
```

### 模块依赖关系

```
Client (主客户端)
├── Database (BoltDB) [必需]
├── Logger [必需]
├── LLM Manager [可选]
├── Browser Manager [可选]
│   ├── Browser
│   └── Recorder/Player
├── MCP Server [可选, 依赖 Browser]
└── Agent Manager [可选, 依赖 MCP + LLM]
```

## 功能模块

### 1. Client (主客户端)

**文件**: `client.go`

**职责**:
- 统一的初始化入口
- 管理所有子模块的生命周期
- 提供配置验证和依赖检查
- 资源清理和优雅退出

**核心类型**:
```go
type Client struct {
    config         *Config
    db             *storage.BoltDB
    llmManager     *llm.Manager
    browserManager *browser.Manager
    mcpServer      *mcp.MCPServer
    agentManager   *agent.AgentManager
    browserClient  *BrowserClient
    scriptClient   *ScriptClient
    agentClient    *AgentClient
}

type Config struct {
    DatabasePath  string
    EnableBrowser bool
    EnableScript  bool
    EnableAgent   bool
    LogLevel      string
    LogOutput     string
    LLMConfig     *LLMConfig
    BrowserConfig *BrowserConfigOptions
}
```

**关键方法**:
- `New(cfg *Config) (*Client, error)`: 创建客户端
- `Browser() *BrowserClient`: 获取浏览器客户端
- `Script() *ScriptClient`: 获取脚本客户端
- `Agent() *AgentClient`: 获取 Agent 客户端
- `Close() error`: 关闭客户端

### 2. BrowserClient (浏览器模块)

**文件**: `browser.go`

**职责**:
- 浏览器生命周期管理
- 页面导航
- Cookie 管理

**核心方法**:
- `Start(ctx context.Context) error`: 启动浏览器
- `Stop() error`: 停止浏览器
- `IsRunning() bool`: 检查运行状态
- `OpenPage(ctx, url) error`: 打开页面
- `SaveCookies(ctx, name, domain) (string, error)`: 保存 Cookie
- `ImportCookies(ctx, cookieID) error`: 导入 Cookie
- `ListCookies() ([]*CookieConfig, error)`: 列出 Cookie
- `GetCookies(cookieID) (*CookieConfig, error)`: 获取 Cookie
- `DeleteCookies(cookieID) error`: 删除 Cookie

**使用示例**:
```go
client.Browser().Start(ctx)
client.Browser().OpenPage(ctx, "https://example.com")
cookieID, _ := client.Browser().SaveCookies(ctx, "site", "https://example.com")
```

### 3. ScriptClient (脚本模块)

**文件**: `script.go`

**职责**:
- 脚本的 CRUD 操作
- 脚本执行
- 执行历史管理

**核心类型**:
```go
type Script struct {
    ID          string
    Name        string
    Description string
    URL         string
    Actions     []models.ScriptAction
    Tags        []string
    Group       string
}

type ScriptExecution struct {
    ID            string
    ScriptID      string
    ScriptName    string
    Status        string
    StartTime     int64
    EndTime       int64
    Duration      int64
    Error         string
    ExtractedData map[string]string
}
```

**核心方法**:
- `Create(ctx, script) (string, error)`: 创建脚本
- `Get(ctx, scriptID) (*Script, error)`: 获取脚本
- `List(ctx) ([]*Script, error)`: 列出脚本
- `Update(ctx, scriptID, script) error`: 更新脚本
- `Delete(ctx, scriptID) error`: 删除脚本
- `Play(ctx, scriptID) (*ScriptExecution, error)`: 执行脚本
- `GetExecution(ctx, executionID) (*ScriptExecution, error)`: 获取执行记录
- `ListExecutions(ctx, scriptID) ([]*ScriptExecution, error)`: 列出执行记录

**使用示例**:
```go
script := &sdk.Script{
    Name: "Test",
    Actions: []models.ScriptAction{
        {Type: "navigate", URL: "https://example.com"},
        {Type: "click", Selector: "#button"},
    },
}
scriptID, _ := client.Script().Create(ctx, script)
result, _ := client.Script().Play(ctx, scriptID)
```

### 4. AgentClient (Agent 模块)

**文件**: `agent.go`

**职责**:
- 会话管理
- 消息发送(流式/非流式)
- LLM 配置管理

**核心类型**:
```go
type AgentSession struct {
    ID        string
    Messages  []AgentMessage
    CreatedAt int64
    UpdatedAt int64
}

type AgentMessage struct {
    ID        string
    Role      string
    Content   string
    Timestamp int64
    ToolCalls []ToolCall
}

type MessageChunk struct {
    Type      string
    Content   string
    ToolCall  *ToolCall
    Error     string
    MessageID string
}
```

**核心方法**:
- `CreateSession(ctx) (string, error)`: 创建会话
- `GetSession(ctx, sessionID) (*AgentSession, error)`: 获取会话
- `ListSessions(ctx) ([]*AgentSession, error)`: 列出会话
- `DeleteSession(ctx, sessionID) error`: 删除会话
- `SendMessage(ctx, sessionID, message) (string, error)`: 发送消息(非流式)
- `SendMessageStream(ctx, sessionID, message, callback) error`: 发送消息(流式)
- `SendMessageStreamReader(ctx, sessionID, message) (io.ReadCloser, error)`: 发送消息(Reader)
- `SetLLMConfig(ctx, config) error`: 设置 LLM 配置
- `GetLLMConfig(ctx) (*LLMConfig, error)`: 获取 LLM 配置

**使用示例**:
```go
sessionID, _ := client.Agent().CreateSession(ctx)
response, _ := client.Agent().SendMessage(ctx, sessionID, "Hello")

// 流式
client.Agent().SendMessageStream(ctx, sessionID, "Tell me a joke", 
    func(chunk *sdk.MessageChunk) {
        fmt.Print(chunk.Content)
    },
)
```

## 实现细节

### 初始化流程

```
1. 验证配置
   ├── 检查必需参数
   ├── 验证依赖关系
   └── 设置默认值

2. 初始化基础组件
   ├── 日志系统
   └── 数据库连接

3. 初始化 LLM Manager
   ├── 从配置加载
   └── 从数据库加载

4. 按需初始化功能模块
   ├── Browser Manager (EnableBrowser=true)
   ├── MCP Server (EnableAgent=true)
   └── Agent Manager (EnableAgent=true)

5. 创建子客户端
   ├── BrowserClient
   ├── ScriptClient
   └── AgentClient
```

### 资源清理流程

```
1. 停止 Agent Manager
2. 停止 MCP Server
3. 停止浏览器
4. 关闭数据库
5. 等待或超时
```

### 错误处理策略

1. **初始化错误**: 立即返回,阻止客户端创建
2. **运行时错误**: 返回 error,由调用方处理
3. **资源清理错误**: 记录日志,继续清理其他资源

### 并发安全

- 所有公共方法都是线程安全的
- 内部使用 mutex 保护共享状态
- 支持多 goroutine 并发调用

## 使用场景

### 场景 1: 纯脚本自动化

```go
client, _ := sdk.New(&sdk.Config{
    DatabasePath: "./data/db",
    EnableBrowser: true,
    EnableScript: true,
    EnableAgent: false,
})
```

**适用于**:
- Web 自动化测试
- 数据爬取
- 定期任务
- RPA 应用

### 场景 2: Agent + 脚本

```go
client, _ := sdk.New(&sdk.Config{
    DatabasePath: "./data/db",
    EnableBrowser: true,
    EnableScript: true,
    EnableAgent: true,
    LLMConfig: &sdk.LLMConfig{...},
})
```

**适用于**:
- 智能自动化
- 对话式操作
- 复杂任务编排
- AI 辅助工作流

### 场景 3: 仅 Agent 对话

```go
client, _ := sdk.New(&sdk.Config{
    DatabasePath: "./data/db",
    EnableBrowser: false,
    EnableScript: false,
    EnableAgent: true,
    LLMConfig: &sdk.LLMConfig{...},
})
```

**适用于**:
- 聊天应用
- 知识问答
- AI 助手
- 对话系统

## 与 Web 服务的区别

| 特性 | Web 服务 | SDK |
|------|---------|-----|
| 部署方式 | 独立服务器 | 嵌入应用 |
| 通信方式 | HTTP/WebSocket | 函数调用 |
| 性能 | 网络延迟 | 直接调用 |
| 集成难度 | 需要 HTTP 客户端 | Go import |
| 资源占用 | 独立进程 | 共享进程 |
| 适用场景 | 多语言、远程访问 | Go 项目、本地集成 |

## 扩展性

### 添加新功能

1. 创建新的客户端类型 (如 `RecorderClient`)
2. 在 `Client` 中添加字段和方法
3. 在 `New()` 中初始化
4. 更新文档

### 添加新的脚本动作

直接使用 `models.ScriptAction`,支持所有现有动作类型。

### 自定义 LLM

通过 `LLMConfig` 配置任意兼容的 LLM 提供商。

## 测试

### 单元测试

```bash
cd backend/sdk
go test -v ./...
```

### 集成测试

运行示例程序:

```bash
cd backend/sdk/examples/basic
go run main.go
```

## 性能考虑

1. **数据库访问**: 使用 BoltDB,适合单机高并发
2. **浏览器启动**: 约 1-2 秒,建议复用
3. **脚本执行**: 取决于操作复杂度
4. **Agent 对话**: 取决于 LLM 响应速度

## 最佳实践

1. **资源管理**: 使用 defer 确保资源释放
2. **上下文控制**: 使用带超时的 context
3. **错误处理**: 检查所有 error 返回值
4. **日志记录**: 设置合适的日志级别
5. **并发控制**: 使用 goroutine 池限制并发数

## 兼容性

- **Go 版本**: 1.21+
- **操作系统**: Linux, macOS, Windows
- **浏览器**: Chrome/Chromium

## 未来计划

- [ ] 支持更多脚本动作类型
- [ ] 添加脚本调试功能
- [ ] 支持分布式执行
- [ ] 提供更多示例
- [ ] 性能优化和监控

## 贡献指南

欢迎贡献代码、文档和示例!

1. Fork 项目
2. 创建特性分支
3. 提交代码
4. 创建 Pull Request

## 许可证

遵循项目主仓库的许可证。
