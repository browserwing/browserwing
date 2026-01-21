# Executor 模块快速开始

## 5 分钟快速上手

### 1. 基本使用

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/browserwing/browserwing/backend/executor"
    "github.com/browserwing/browserwing/services/browser"
)

func main() {
    ctx := context.Background()
    
    // 创建浏览器管理器
    browserMgr := browser.NewManager(cfg, db, llmMgr)
    browserMgr.Start(ctx)
    defer browserMgr.Stop()
    
    // 创建 Executor
    exec := executor.NewExecutor(browserMgr)
    
    // 打开页面
    exec.Navigate(ctx, "https://example.com", nil)
    
    // 智能交互
    exec.ClickByLabel(ctx, "Login")
    exec.TypeByLabel(ctx, "Username", "myuser")
    
    fmt.Println("操作完成！")
}
```

### 2. 获取页面语义树

```go
// 获取语义树
tree, _ := exec.GetSemanticTree(ctx)

// 打印所有可点击元素
for _, node := range tree.GetClickableElements() {
    fmt.Printf("按钮: %s\n", node.Label)
}

// 查找特定元素
loginBtn := tree.FindElementByLabel("Login")
if loginBtn != nil {
    fmt.Printf("找到登录按钮: %s\n", loginBtn.Selector)
}
```

### 3. 数据提取

```go
// 提取单个元素
result, _ := exec.Extract(ctx, &executor.ExtractOptions{
    Selector: ".title",
    Type:     "text",
})
fmt.Println(result.Data["result"])

// 提取多个元素
result, _ = exec.Extract(ctx, &executor.ExtractOptions{
    Selector: ".product",
    Type:     "text",
    Multiple: true,
})
```

### 4. 作为 MCP 工具使用

```go
// 在 MCP Server 中注册
registry := executor.NewMCPToolRegistry(exec, mcpServer)
registry.RegisterAllTools()

// 现在可以通过 MCP 调用
// Claude: "请帮我打开 example.com 并点击登录"
```

## 常见操作速查

### 导航
```go
exec.Navigate(ctx, "https://example.com", nil)
exec.GoBack(ctx)
exec.GoForward(ctx)
exec.Reload(ctx)
```

### 交互
```go
exec.Click(ctx, "button.submit", nil)
exec.Type(ctx, "input[name='email']", "user@example.com", nil)
exec.Select(ctx, "select[name='country']", "China", nil)
exec.Hover(ctx, ".menu-item")
```

### 智能交互（推荐）
```go
exec.ClickByLabel(ctx, "提交")
exec.TypeByLabel(ctx, "邮箱", "user@example.com")
exec.SelectByLabel(ctx, "国家", "中国")
```

### 等待
```go
exec.WaitFor(ctx, ".loading", &executor.WaitForOptions{
    State: "hidden",
})
```

### 截图
```go
result, _ := exec.Screenshot(ctx, &executor.ScreenshotOptions{
    FullPage: true,
})
```

## 完整示例：自动登录

```go
func autoLogin(exec *executor.Executor) error {
    ctx := context.Background()
    
    // 1. 打开登录页面
    _, err := exec.Navigate(ctx, "https://example.com/login", nil)
    if err != nil {
        return err
    }
    
    // 2. 等待页面加载
    exec.WaitFor(ctx, "input[name='username']", nil)
    
    // 3. 填写表单
    exec.TypeByLabel(ctx, "用户名", "myuser")
    exec.TypeByLabel(ctx, "密码", "mypassword")
    
    // 4. 点击登录
    exec.ClickByLabel(ctx, "登录")
    
    // 5. 等待登录成功
    exec.WaitFor(ctx, ".dashboard", &executor.WaitForOptions{
        State: "visible",
    })
    
    fmt.Println("登录成功！")
    return nil
}
```

## 完整示例：数据采集

```go
func scrapeProducts(exec *executor.Executor) ([]Product, error) {
    ctx := context.Background()
    
    // 1. 打开产品列表页
    exec.Navigate(ctx, "https://example.com/products", nil)
    
    // 2. 滚动加载更多
    exec.ScrollToBottom(ctx)
    time.Sleep(2 * time.Second)
    
    // 3. 提取产品信息
    result, err := exec.Extract(ctx, &executor.ExtractOptions{
        Selector: ".product-item",
        Multiple: true,
        Fields:   []string{"text", "href"},
    })
    if err != nil {
        return nil, err
    }
    
    // 4. 解析结果
    products := []Product{}
    if items, ok := result.Data["result"].([]map[string]interface{}); ok {
        for _, item := range items {
            products = append(products, Product{
                Name: item["text"].(string),
                URL:  item["href"].(string),
            })
        }
    }
    
    return products, nil
}
```

## 完整示例：批量操作

```go
func batchTest(exec *executor.Executor) error {
    ctx := context.Background()
    
    operations := []executor.Operation{
        {
            Type: "navigate",
            Params: map[string]interface{}{
                "url": "https://example.com",
            },
            StopOnError: true,
        },
        {
            Type: "type",
            Params: map[string]interface{}{
                "identifier": "搜索",
                "text":       "browserwing",
            },
            StopOnError: true,
        },
        {
            Type: "click",
            Params: map[string]interface{}{
                "identifier": "搜索按钮",
            },
            StopOnError: true,
        },
        {
            Type: "wait",
            Params: map[string]interface{}{
                "identifier": ".results",
            },
            StopOnError: false,
        },
        {
            Type: "screenshot",
            Params:      map[string]interface{}{},
            StopOnError: false,
        },
    }
    
    result, err := exec.ExecuteBatch(ctx, operations)
    if err != nil {
        return err
    }
    
    fmt.Printf("成功: %d, 失败: %d\n", result.Success, result.Failed)
    return nil
}
```

## MCP 工具使用示例

### 在 Claude Desktop 中使用

1. 配置 `claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "browserwing": {
      "command": "/path/to/browserwing",
      "args": ["--mcp"]
    }
  }
}
```

2. 在 Claude 中使用：
```
用户: 请帮我打开 https://github.com 并搜索 "browserwing"

Claude 会自动调用:
1. browser_navigate(url="https://github.com")
2. browser_get_semantic_tree() 
3. browser_type(identifier="搜索框", text="browserwing")
4. browser_click(identifier="搜索按钮")
```

### 通过 HTTP 调用 MCP 工具

```bash
curl -X POST http://localhost:8080/api/v1/mcp/message \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "browser_navigate",
      "arguments": {
        "url": "https://example.com"
      }
    }
  }'
```

## 调试技巧

### 1. 高亮显示元素
```go
exec.HighlightElementByLabel(ctx, "Login")
time.Sleep(2 * time.Second) // 观察高亮效果
```

### 2. 打印语义树
```go
tree, _ := exec.GetSemanticTree(ctx)
fmt.Println(tree.SerializeToSimpleText())
```

### 3. 检查页面信息
```go
result, _ := exec.GetPageInfo(ctx)
fmt.Printf("URL: %s\n", result.Data["url"])
fmt.Printf("Title: %s\n", result.Data["title"])
```

### 4. 截图调试
```go
result, _ := exec.Screenshot(ctx, &executor.ScreenshotOptions{
    FullPage: true,
})
os.WriteFile("debug.png", result.Data["data"].([]byte), 0644)
```

## 常见问题

### Q: 元素找不到怎么办？
```go
// 1. 先获取语义树查看所有元素
tree, _ := exec.GetSemanticTree(ctx)
fmt.Println(tree.SerializeToSimpleText())

// 2. 使用更模糊的匹配
node := tree.FindElementByLabel("登") // 只匹配部分文字

// 3. 等待元素出现
exec.WaitFor(ctx, "button", &executor.WaitForOptions{
    State:   "visible",
    Timeout: 10 * time.Second,
})
```

### Q: 页面加载慢怎么办？
```go
// 使用更长的超时
exec.Navigate(ctx, url, &executor.NavigateOptions{
    WaitUntil: "networkidle",
    Timeout:   60 * time.Second,
})
```

### Q: 如何处理动态内容？
```go
// 等待内容加载
exec.WaitFor(ctx, ".dynamic-content", nil)

// 或者使用批量操作的重试机制
```

## 下一步

- 📖 阅读 [完整 API 文档](./README.md)
- 🔧 查看 [集成指南](./INTEGRATION.md)
- 📝 查看 [更多示例](./examples.go)
- 📊 阅读 [项目总结](./SUMMARY.md)

## 获取帮助

- 查看文档中的示例
- 使用 `tree.SerializeToSimpleText()` 了解页面结构
- 启用调试模式查看详细日志
- 提交 Issue 获取支持

---

开始使用 Executor 模块，让浏览器自动化变得更智能！🚀

