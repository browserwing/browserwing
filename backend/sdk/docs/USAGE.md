# BrowserWing SDK 使用指南

## 安装

在你的 Go 项目中导入 SDK:

```go
import "github.com/browserwing/browserwing/sdk"
```

## 快速开始

### 最简单的例子

```go
package main

import (
    "context"
    "log"
    "github.com/browserwing/browserwing/sdk"
)

func main() {
    // 创建客户端
    client, err := sdk.New(&sdk.Config{
        DatabasePath: "./data/browserwing.db",
        EnableBrowser: true,
        EnableScript: true,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 使用功能
    ctx := context.Background()
    client.Browser().Start(ctx)
    defer client.Browser().Stop()
}
```

## 配置选项

### Config 结构

```go
type Config struct {
    // 必需
    DatabasePath string // 数据库路径
    
    // 功能开关
    EnableBrowser bool // 启用浏览器
    EnableScript  bool // 启用脚本(需要浏览器)
    EnableAgent   bool // 启用 Agent(需要 LLM)
    
    // 可选
    LogLevel      string           // debug/info/warn/error
    LogOutput     string           // stdout/stderr/文件路径
    LLMConfig     *LLMConfig       // LLM 配置
    BrowserConfig *BrowserConfigOptions // 浏览器配置
}
```

## 功能模块

### 1. 浏览器管理 (BrowserClient)

通过 `client.Browser()` 访问。

#### 启动和停止

```go
// 启动浏览器
err := client.Browser().Start(ctx)

// 停止浏览器
err := client.Browser().Stop()

// 检查状态
isRunning := client.Browser().IsRunning()
```

#### 页面访问

```go
// 打开页面
err := client.Browser().OpenPage(ctx, "https://example.com")
```

#### Cookie 管理

```go
// 保存 Cookie
cookieID, err := client.Browser().SaveCookies(ctx, "site-name", "https://example.com")

// 导入 Cookie
err := client.Browser().ImportCookies(ctx, cookieID)

// 列出所有 Cookie
cookies, err := client.Browser().ListCookies()

// 获取指定 Cookie
cookie, err := client.Browser().GetCookies(cookieID)

// 删除 Cookie
err := client.Browser().DeleteCookies(cookieID)
```

### 2. 脚本管理 (ScriptClient)

通过 `client.Script()` 访问。

#### 创建脚本

```go
script := &sdk.Script{
    Name: "My Script",
    Description: "Script description",
    URL: "https://example.com",
    Actions: []models.ScriptAction{
        {
            Type: "navigate",
            URL: "https://example.com",
        },
        {
            Type: "click",
            Selector: "#button",
        },
    },
    Tags: []string{"tag1", "tag2"},
    Group: "group-name",
}

scriptID, err := client.Script().Create(ctx, script)
```

#### 脚本操作

```go
// 获取脚本
script, err := client.Script().Get(ctx, scriptID)

// 列出所有脚本
scripts, err := client.Script().List(ctx)

// 更新脚本
script.Name = "Updated Name"
err := client.Script().Update(ctx, scriptID, script)

// 删除脚本
err := client.Script().Delete(ctx, scriptID)
```

#### 执行脚本

```go
// 执行脚本
result, err := client.Script().Play(ctx, scriptID)

// 查看结果
log.Printf("Status: %s", result.Status)
log.Printf("Duration: %d ms", result.Duration)
log.Printf("Extracted data: %v", result.ExtractedData)
```

#### 执行历史

```go
// 获取执行记录
execution, err := client.Script().GetExecution(ctx, executionID)

// 列出所有执行记录
executions, err := client.Script().ListExecutions(ctx, scriptID)

// 列出所有脚本的执行记录
allExecutions, err := client.Script().ListExecutions(ctx, "")
```

### 3. Agent 对话 (AgentClient)

通过 `client.Agent()` 访问。需要在配置中启用 Agent 并提供 LLM 配置。

#### 会话管理

```go
// 创建会话
sessionID, err := client.Agent().CreateSession(ctx)

// 获取会话
session, err := client.Agent().GetSession(ctx, sessionID)

// 列出所有会话
sessions, err := client.Agent().ListSessions(ctx)

// 删除会话
err := client.Agent().DeleteSession(ctx, sessionID)
```

#### 发送消息 - 非流式

```go
response, err := client.Agent().SendMessage(ctx, sessionID, "你好!")
log.Printf("Response: %s", response)
```

#### 发送消息 - 流式

```go
err := client.Agent().SendMessageStream(ctx, sessionID, "讲个笑话", 
    func(chunk *sdk.MessageChunk) {
        switch chunk.Type {
        case "message":
            fmt.Print(chunk.Content)
        case "tool_call":
            fmt.Printf("\n[Tool: %s]", chunk.ToolCall.ToolName)
        case "error":
            fmt.Printf("\nError: %s", chunk.Error)
        case "done":
            fmt.Println("\nDone")
        }
    },
)
```

#### 发送消息 - Reader 接口

```go
reader, err := client.Agent().SendMessageStreamReader(ctx, sessionID, "你好")
if err != nil {
    log.Fatal(err)
}
defer reader.Close()

// 使用 json.Decoder 读取流
decoder := json.NewDecoder(reader)
for {
    var chunk sdk.MessageChunk
    if err := decoder.Decode(&chunk); err == io.EOF {
        break
    } else if err != nil {
        log.Fatal(err)
    }
    fmt.Print(chunk.Content)
}
```

#### LLM 配置

```go
// 设置 LLM 配置
err := client.Agent().SetLLMConfig(ctx, &sdk.LLMConfig{
    Provider: "openai",
    APIKey: "sk-xxx",
    Model: "gpt-4",
    BaseURL: "https://api.openai.com/v1",
})

// 获取当前 LLM 配置
config, err := client.Agent().GetLLMConfig(ctx)
```

## 脚本动作类型

### 导航

```go
{
    Type: "navigate",
    URL: "https://example.com",
}
```

### 点击

```go
{
    Type: "click",
    Selector: "#button",
    Description: "点击按钮",
}
```

### 输入

```go
{
    Type: "input",
    Selector: "#email",
    Value: "user@example.com",
    Description: "输入邮箱",
}
```

### 等待

```go
{
    Type: "wait",
    Duration: 2000, // 毫秒
}
```

### 提取文本

```go
{
    Type: "extract_text",
    Selector: "h1",
    ExtractType: "text",
    VariableName: "title",
    Description: "提取标题",
}
```

### 提取属性

```go
{
    Type: "extract_attribute",
    Selector: "img",
    ExtractType: "attribute",
    AttributeName: "src",
    VariableName: "image_url",
}
```

### 执行 JavaScript

```go
{
    Type: "execute_js",
    JSCode: "return document.title;",
    VariableName: "page_title",
}
```

### 键盘操作

```go
{
    Type: "keyboard",
    Key: "Enter",
}
```

### 滚动

```go
{
    Type: "scroll",
    ScrollX: 0,
    ScrollY: 500,
}
```

## 使用场景

### 场景 1: Web 自动化测试

```go
client, _ := sdk.New(&sdk.Config{
    DatabasePath: "./data/test.db",
    EnableBrowser: true,
    EnableScript: true,
})
defer client.Close()

ctx := context.Background()
client.Browser().Start(ctx)
defer client.Browser().Stop()

// 创建测试脚本
script := &sdk.Script{
    Name: "Login Test",
    Actions: []models.ScriptAction{
        {Type: "navigate", URL: "https://app.example.com/login"},
        {Type: "input", Selector: "#username", Value: "testuser"},
        {Type: "input", Selector: "#password", Value: "testpass"},
        {Type: "click", Selector: "#login-btn"},
        {Type: "wait", Duration: 2000},
        {Type: "extract_text", Selector: ".welcome", VariableName: "welcome_msg"},
    },
}

scriptID, _ := client.Script().Create(ctx, script)
result, _ := client.Script().Play(ctx, scriptID)

if result.Status == "success" {
    log.Println("Login test passed")
}
```

### 场景 2: 数据爬取

```go
// 爬取多个页面
urls := []string{
    "https://example.com/page1",
    "https://example.com/page2",
    "https://example.com/page3",
}

for _, url := range urls {
    script := &sdk.Script{
        Name: "Scrape " + url,
        Actions: []models.ScriptAction{
            {Type: "navigate", URL: url},
            {Type: "wait", Duration: 1000},
            {Type: "extract_text", Selector: "h1", VariableName: "title"},
            {Type: "extract_text", Selector: ".content", VariableName: "content"},
        },
    }
    
    scriptID, _ := client.Script().Create(ctx, script)
    result, _ := client.Script().Play(ctx, scriptID)
    
    // 处理抓取的数据
    processData(result.ExtractedData)
}
```

### 场景 3: 定期任务

```go
// 每小时检查一次
ticker := time.NewTicker(1 * time.Hour)
defer ticker.Stop()

for range ticker.C {
    result, err := client.Script().Play(ctx, monitorScriptID)
    if err != nil {
        log.Printf("Monitor failed: %v", err)
        continue
    }
    
    // 检查结果并发送通知
    if result.ExtractedData["status"] != "ok" {
        sendAlert(result.ExtractedData)
    }
}
```

### 场景 4: 与 Agent 结合

```go
client, _ := sdk.New(&sdk.Config{
    DatabasePath: "./data/app.db",
    EnableBrowser: true,
    EnableScript: true,
    EnableAgent: true,
    LLMConfig: &sdk.LLMConfig{
        Provider: "openai",
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model: "gpt-4",
    },
})

// 创建会话
sessionID, _ := client.Agent().CreateSession(ctx)

// Agent 可以调用浏览器脚本
response, _ := client.Agent().SendMessage(ctx, sessionID, 
    "帮我访问 example.com 并提取页面标题")

log.Println(response)
```

## 错误处理

SDK 中的大部分方法都返回 error,建议统一处理:

```go
if err := client.Browser().Start(ctx); err != nil {
    log.Fatalf("Failed to start browser: %v", err)
}

// 或使用更优雅的错误处理
if err := client.Script().Play(ctx, scriptID); err != nil {
    switch {
    case errors.Is(err, sdk.ErrBrowserNotRunning):
        log.Println("Please start browser first")
    case errors.Is(err, sdk.ErrScriptNotFound):
        log.Println("Script not found")
    default:
        log.Printf("Unknown error: %v", err)
    }
}
```

## 最佳实践

### 1. 资源清理

始终使用 defer 确保资源被正确释放:

```go
client, err := sdk.New(config)
if err != nil {
    log.Fatal(err)
}
defer client.Close() // 重要!

if err := client.Browser().Start(ctx); err != nil {
    log.Fatal(err)
}
defer client.Browser().Stop() // 重要!
```

### 2. 上下文管理

使用带超时的 context:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := client.Script().Play(ctx, scriptID)
```

### 3. 错误恢复

为长期运行的任务添加错误恢复:

```go
for {
    err := client.Script().Play(ctx, scriptID)
    if err != nil {
        log.Printf("Error: %v, retrying in 1 minute", err)
        time.Sleep(1 * time.Minute)
        continue
    }
    break
}
```

### 4. 并发安全

SDK 的大部分操作是线程安全的,可以在多个 goroutine 中使用:

```go
var wg sync.WaitGroup
for _, scriptID := range scriptIDs {
    wg.Add(1)
    go func(id string) {
        defer wg.Done()
        client.Script().Play(ctx, id)
    }(scriptID)
}
wg.Wait()
```

## 示例代码

完整的示例代码请参考:

- `examples/basic/main.go` - 基础功能示例
- `examples/agent/main.go` - Agent 对话示例  
- `examples/advanced/main.go` - 高级场景示例

## 常见问题

### Q: 如何只使用脚本功能,不使用 Agent?

A: 在配置中设置 `EnableAgent: false`:

```go
client, _ := sdk.New(&sdk.Config{
    DatabasePath: "./data/browserwing.db",
    EnableBrowser: true,
    EnableScript: true,
    EnableAgent: false, // 不启用 Agent
})
```

### Q: 浏览器启动失败怎么办?

A: 检查以下几点:
1. 确保系统已安装 Chrome/Chromium
2. 检查端口是否被占用
3. 查看日志输出的详细错误信息

### Q: 如何在 Docker 中使用?

A: 需要安装浏览器依赖:

```dockerfile
FROM golang:1.21

RUN apt-get update && apt-get install -y \
    chromium \
    chromium-driver

COPY . /app
WORKDIR /app
RUN go build -o myapp

CMD ["./myapp"]
```

### Q: 脚本执行超时怎么办?

A: 使用带超时的 context:

```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

result, err := client.Script().Play(ctx, scriptID)
```

## 获取帮助

- 查看项目文档: `/docs`
- 提交 Issue: GitHub Issues
- 加入社区讨论

## 许可证

本 SDK 遵循项目主仓库的许可证。
