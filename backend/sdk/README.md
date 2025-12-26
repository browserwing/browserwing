# BrowserWing SDK

BrowserWing SDK 提供了一套完整的 Go 语言接口,允许其他 Go 项目以编程方式使用 BrowserWing 的核心功能。

## 核心能力

### 1. 初始化与配置
- 灵活的初始化选项,支持选择性启用模块
- 数据库配置
- 日志配置
- LLM 配置(可选)
- 浏览器配置(可选)

### 2. 浏览器管理
- 启动/停止浏览器
- 访问指定页面
- Cookie 管理(保存/导入)
- 获取浏览器状态

### 3. 脚本管理
- 创建/更新/删除脚本
- 列出所有脚本
- 执行脚本(播放)
- 获取脚本执行结果
- 获取执行历史

### 4. Agent 对话
- 创建会话
- 发送消息(支持流式和非流式)
- 获取会话历史
- 删除会话
- 配置 LLM

## 使用示例

### 基础初始化(仅使用脚本功能)

```go
package main

import (
    "context"
    "log"
    "github.com/browserwing/browserwing/sdk"
)

func main() {
    // 仅启用浏览器和脚本功能
    client, err := sdk.New(&sdk.Config{
        DatabasePath: "./data/browserwing.db",
        EnableBrowser: true,
        EnableScript: true,
        EnableAgent: false, // 不启用 Agent
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 启动浏览器
    ctx := context.Background()
    if err := client.Browser().Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Browser().Stop()

    // 执行脚本
    result, err := client.Script().Play(ctx, "script-id")
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Script executed: %+v", result)
}
```

### 完整功能(包含 Agent)

```go
package main

import (
    "context"
    "log"
    "github.com/browserwing/browserwing/sdk"
)

func main() {
    // 启用所有功能
    client, err := sdk.New(&sdk.Config{
        DatabasePath: "./data/browserwing.db",
        EnableBrowser: true,
        EnableScript: true,
        EnableAgent: true,
        LLMConfig: &sdk.LLMConfig{
            Provider: "openai",
            APIKey: "your-api-key",
            Model: "gpt-4",
            BaseURL: "https://api.openai.com/v1",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    // Agent 对话示例
    sessionID, err := client.Agent().CreateSession(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // 流式对话
    err = client.Agent().SendMessageStream(ctx, sessionID, "帮我搜索今天的新闻", func(chunk string) {
        log.Print(chunk)
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

### 浏览器和 Cookie 管理

```go
// 启动浏览器并访问页面
ctx := context.Background()
if err := client.Browser().Start(ctx); err != nil {
    log.Fatal(err)
}

// 访问页面
if err := client.Browser().OpenPage(ctx, "https://example.com"); err != nil {
    log.Fatal(err)
}

// 保存 Cookie
cookieID, err := client.Browser().SaveCookies(ctx, "example-site", "https://example.com")
if err != nil {
    log.Fatal(err)
}

// 导入 Cookie
if err := client.Browser().ImportCookies(ctx, cookieID); err != nil {
    log.Fatal(err)
}
```

### 脚本管理

```go
// 创建脚本
script := &sdk.Script{
    Name: "Test Script",
    Description: "A test automation script",
    URL: "https://example.com",
    Actions: []sdk.ScriptAction{
        {
            Type: "navigate",
            URL: "https://example.com",
        },
        {
            Type: "click",
            Selector: "#button",
        },
    },
}

scriptID, err := client.Script().Create(ctx, script)
if err != nil {
    log.Fatal(err)
}

// 列出所有脚本
scripts, err := client.Script().List(ctx)
if err != nil {
    log.Fatal(err)
}

// 执行脚本
result, err := client.Script().Play(ctx, scriptID)
if err != nil {
    log.Fatal(err)
}

// 更新脚本
script.Name = "Updated Script"
if err := client.Script().Update(ctx, scriptID, script); err != nil {
    log.Fatal(err)
}

// 删除脚本
if err := client.Script().Delete(ctx, scriptID); err != nil {
    log.Fatal(err)
}
```

## 架构设计

SDK 采用模块化设计,主要包含以下模块:

- **Client**: 主客户端,管理所有子模块
- **BrowserClient**: 浏览器管理模块
- **ScriptClient**: 脚本管理模块
- **AgentClient**: Agent 对话模块

每个模块可以独立启用或禁用,以适应不同的使用场景。
