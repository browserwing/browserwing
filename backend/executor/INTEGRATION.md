# Executor 模块集成指南

本文档说明如何将 Executor 模块集成到现有的 browserwing 项目中，并作为 MCP 工具提供给外部调用。

## 架构概览

```
┌─────────────────┐
│   AI Agent      │
│   (Claude等)    │
└────────┬────────┘
         │ MCP Protocol
         ▼
┌─────────────────┐
│   MCP Server    │
│  (mcp/server.go)│
└────────┬────────┘
         │
         ▼
┌─────────────────┐      ┌──────────────────┐
│    Executor     │◄─────┤ Browser Manager  │
│   (executor/)   │      │  (services/browser)│
└─────────────────┘      └──────────────────┘
         │
         ▼
┌─────────────────┐
│   Rod Browser   │
│   (go-rod/rod)  │
└─────────────────┘
```

## 集成步骤

### 1. 在 MCP Server 中初始化 Executor

修改 `backend/mcp/server.go`，添加 Executor 支持：

```go
package mcp

import (
	"github.com/browserwing/browserwing/backend/executor"
	// ... 其他导入
)

type MCPServer struct {
	// ... 现有字段
	executor *executor.Executor // 添加 Executor 实例
	toolRegistry *executor.MCPToolRegistry // 添加工具注册表
}

func NewMCPServer(storage *storage.BoltDB, browserMgr *browser.Manager) *MCPServer {
	ctx, cancel := context.WithCancel(context.Background())

	s := &MCPServer{
		storage:       storage,
		browserMgr:    browserMgr,
		scripts:       make(map[string]*models.Script),
		scriptsByName: make(map[string]*models.Script),
		ctx:           ctx,
		cancel:        cancel,
	}

	// 创建 mcp-go server
	s.mcpServer = server.NewMCPServer(
		"browserwing",
		"0.0.1",
		server.WithToolCapabilities(true),
	)

	// 初始化 Executor
	s.executor = executor.NewExecutor(browserMgr)
	
	// 创建工具注册表
	s.toolRegistry = executor.NewMCPToolRegistry(s.executor, s.mcpServer)

	// ... 其他初始化代码

	return s
}
```

### 2. 注册 Executor 工具

在 `Start()` 方法中注册 Executor 工具：

```go
func (s *MCPServer) Start() error {
	logger.Info(s.ctx, "MCP server started")

	// 加载所有标记为 MCP 命令的脚本
	if err := s.loadMCPScripts(); err != nil {
		return fmt.Errorf("failed to load MCP scripts: %w", err)
	}

	// 注册所有脚本为工具
	if err := s.registerAllTools(); err != nil {
		return fmt.Errorf("failed to register tools: %w", err)
	}

	// 注册 Executor 工具
	if err := s.toolRegistry.RegisterAllTools(); err != nil {
		return fmt.Errorf("failed to register executor tools: %w", err)
	}

	logger.Info(s.ctx, "Registered %d executor tools", len(s.toolRegistry.GetToolMetadata()))

	return nil
}
```

### 3. 在主程序中使用

修改 `cmd/main.go`（或你的主程序文件）：

```go
package main

import (
	"context"
	
	"github.com/browserwing/browserwing/backend/executor"
	"github.com/browserwing/browserwing/backend/mcp"
	"github.com/browserwing/browserwing/services/browser"
	// ... 其他导入
)

func main() {
	// ... 初始化配置、数据库等

	// 创建浏览器管理器
	browserMgr := browser.NewManager(cfg, db, llmMgr)

	// 创建 MCP 服务器（会自动初始化 Executor）
	mcpServer := mcp.NewMCPServer(db, browserMgr)
	
	// 启动 MCP 服务器
	if err := mcpServer.Start(); err != nil {
		log.Fatal(err)
	}

	// 启动 HTTP 服务器
	if err := mcpServer.StartStreamableHTTPServer(":8080"); err != nil {
		log.Fatal(err)
	}

	// ... 其他启动代码
}
```

## 可用的 MCP 工具

集成后，以下工具将自动注册到 MCP 服务器：

### 导航类
- `browser_navigate` - 导航到 URL
- `browser_scroll` - 滚动页面

### 交互类
- `browser_click` - 点击元素
- `browser_type` - 输入文本
- `browser_select` - 选择下拉选项

### 数据类
- `browser_extract` - 提取数据
- `browser_get_page_info` - 获取页面信息
- `browser_get_semantic_tree` - 获取语义树

### 捕获类
- `browser_screenshot` - 截图

### 同步类
- `browser_wait_for` - 等待元素状态

## 使用示例

### 通过 MCP 客户端调用

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "browser_navigate",
    "arguments": {
      "url": "https://example.com",
      "wait_until": "load"
    }
  }
}
```

### 在 Claude Desktop 中使用

配置 `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "browserwing": {
      "command": "browserwing",
      "args": ["--mcp"],
      "env": {}
    }
  }
}
```

然后在 Claude 中可以这样使用：

```
请帮我打开 https://example.com 并点击"登录"按钮
```

Claude 会自动调用：
1. `browser_navigate` 打开页面
2. `browser_get_semantic_tree` 获取页面元素
3. `browser_click` 点击登录按钮

## 高级用法

### 1. 自定义工具

你可以添加自己的工具到注册表：

```go
// 在 mcp_tools.go 中添加新工具
func (r *MCPToolRegistry) registerCustomTool() error {
	tool := mcpgo.NewTool(
		"browser_custom_action",
		mcpgo.WithDescription("Your custom action"),
		mcpgo.WithString("param1", mcpgo.Required(), mcpgo.Description("Parameter 1")),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		// 实现你的逻辑
		return mcpgo.NewToolResultText("Success"), nil
	}

	r.mcpServer.AddTool(tool, handler)
	return nil
}
```

### 2. 直接使用 Executor

除了通过 MCP，你也可以在代码中直接使用 Executor：

```go
// 创建 Executor
exec := executor.NewExecutor(browserMgr)

// 导航
exec.Navigate(ctx, "https://example.com", nil)

// 智能交互
exec.TypeByLabel(ctx, "Username", "myuser")
exec.ClickByLabel(ctx, "Login")

// 获取语义树
tree, _ := exec.GetSemanticTree(ctx)
fmt.Println(tree.SerializeToSimpleText())
```

### 3. 批量操作

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
			"identifier": "Username",
			"text":       "myuser",
		},
	},
	{
		Type: "click",
		Params: map[string]interface{}{
			"identifier": "Login",
		},
	},
}

result, _ := exec.ExecuteBatch(ctx, operations)
```

## 与现有功能的对比

### Executor vs 脚本录制回放

| 特性 | 脚本录制回放 | Executor |
|------|-------------|----------|
| 使用方式 | 录制 → 回放 | 编程式 API |
| 灵活性 | 固定流程 | 动态交互 |
| 语义理解 | 基于选择器 | 深度语义 |
| MCP 支持 | 通过脚本 | 原生支持 |
| 适用场景 | 重复任务 | AI Agent |

### 建议使用场景

- **使用脚本录制回放**：
  - 固定的、重复的自动化任务
  - 需要保存和分享的工作流
  - 不需要动态决策的场景

- **使用 Executor**：
  - AI Agent 驱动的自动化
  - 需要根据页面内容动态决策
  - 需要通过 MCP 提供服务
  - 需要编程式控制的场景

## 性能优化

### 1. 缓存语义树

```go
type CachedExecutor struct {
	*executor.Executor
	treeCache *executor.SemanticTree
	cacheTime time.Time
}

func (e *CachedExecutor) GetSemanticTree(ctx context.Context) (*executor.SemanticTree, error) {
	// 如果缓存未过期，返回缓存
	if time.Since(e.cacheTime) < 5*time.Second && e.treeCache != nil {
		return e.treeCache, nil
	}
	
	// 重新获取
	tree, err := e.Executor.GetSemanticTree(ctx)
	if err != nil {
		return nil, err
	}
	
	e.treeCache = tree
	e.cacheTime = time.Now()
	return tree, nil
}
```

### 2. 并发操作

```go
// 并发获取多个页面的信息
var wg sync.WaitGroup
results := make([]interface{}, len(urls))

for i, url := range urls {
	wg.Add(1)
	go func(idx int, u string) {
		defer wg.Done()
		exec := executor.NewExecutor(browserMgr)
		exec.Navigate(ctx, u, nil)
		tree, _ := exec.GetSemanticTree(ctx)
		results[idx] = tree
	}(i, url)
}

wg.Wait()
```

## 故障排查

### 1. 浏览器未启动

```go
if !exec.IsReady() {
	if err := browserMgr.Start(ctx); err != nil {
		return err
	}
	exec.WaitUntilReady(ctx, 30*time.Second)
}
```

### 2. 元素未找到

```go
// 使用语义树查找
tree, _ := exec.GetSemanticTree(ctx)
node := tree.FindElementByLabel("Login")
if node == nil {
	// 尝试其他方式
	node = tree.FindElementByLabel("Sign In")
}
```

### 3. 页面加载超时

```go
result, err := exec.Navigate(ctx, url, &executor.NavigateOptions{
	WaitUntil: "networkidle",
	Timeout:   60 * time.Second,
})
```

## 最佳实践

1. **始终检查 Executor 是否就绪**
   ```go
   if !exec.IsReady() {
       return fmt.Errorf("executor not ready")
   }
   ```

2. **使用语义标识而非选择器**
   ```go
   // 好
   exec.ClickByLabel(ctx, "Submit")
   
   // 不好
   exec.Click(ctx, "button.btn-submit-form-123", nil)
   ```

3. **适当使用等待**
   ```go
   exec.WaitFor(ctx, "Loading indicator", &executor.WaitForOptions{
       State: "hidden",
   })
   ```

4. **处理错误**
   ```go
   result, err := exec.Click(ctx, "Button", nil)
   if err != nil {
       logger.Error(ctx, "Click failed: %v", err)
       return err
   }
   if !result.Success {
       logger.Warn(ctx, "Click unsuccessful: %s", result.Message)
   }
   ```

## 下一步

- 查看 [README.md](./README.md) 了解更多 API 文档
- 查看 [examples.go](./examples.go) 了解更多使用示例
- 参考 [playwright-mcp](https://github.com/executeautomation/playwright-mcp) 了解 MCP 最佳实践
- 参考 [agent-browser](https://github.com/vercel-labs/agent-browser) 了解语义化浏览器自动化

