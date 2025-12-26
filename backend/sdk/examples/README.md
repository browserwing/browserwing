# BrowserWing SDK Examples

这里提供了三个完整的示例,展示如何使用 BrowserWing SDK。

## 示例列表

### 1. Basic Example (基础示例)

**文件**: `basic/main.go`

**功能**:
- 启动浏览器
- 访问页面
- 创建脚本
- 执行脚本
- 查看执行历史

**运行**:
```bash
cd basic
go run main.go
```

### 2. Agent Example (Agent 对话示例)

**文件**: `agent/main.go`

**功能**:
- 创建 Agent 会话
- 发送消息(非流式)
- 发送消息(流式)
- 查看会话历史
- 使用浏览器工具(可选)

**配置**:
⚠️ 需要配置 LLM API Key! 编辑 `agent/main.go`,修改:
```go
LLMConfig: &sdk.LLMConfig{
    Provider: "openai",
    APIKey:   "your-api-key-here", // 替换为实际的 API Key
    Model:    "gpt-4",
    BaseURL:  "https://api.openai.com/v1",
},
```

**运行**:
```bash
cd agent
go run main.go
```

### 3. Advanced Example (高级示例)

**文件**: `advanced/main.go`

**功能**:
- 数据抓取
- 批量访问页面
- 定期监控
- 脚本管理
- 执行历史分析

**运行**:
```bash
cd advanced
go run main.go
```

## 准备工作

### 1. 确保浏览器已安装

SDK 需要 Chrome 或 Chromium 浏览器:

```bash
# Ubuntu/Debian
sudo apt-get install chromium-browser

# macOS
brew install --cask google-chrome

# 或设置环境变量指向浏览器
export CHROME_BIN_PATH=/path/to/chrome
```

### 2. 初始化依赖

每个示例都需要先初始化依赖:

```bash
cd basic  # 或 agent/advanced
go mod tidy
```

### 3. 创建数据目录

示例会在当前目录创建 `data` 文件夹存储数据库:

```bash
mkdir -p data
```

## 运行所有示例

```bash
# 基础示例
cd basic && go run main.go && cd ..

# Agent 示例 (需要配置 API Key)
cd agent && go run main.go && cd ..

# 高级示例
cd advanced && go run main.go && cd ..
```

## 常见问题

### Q: 浏览器启动失败

**A**: 检查浏览器是否已安装:
```bash
which google-chrome
which chromium-browser
```

如果没有,请先安装浏览器。

### Q: go.mod 报错

**A**: 确保在正确的目录运行:
```bash
cd examples/basic  # 在示例目录中
go mod tidy
go run main.go
```

### Q: Agent 示例无法运行

**A**: 需要配置有效的 LLM API Key:
1. 打开 `agent/main.go`
2. 找到 `LLMConfig` 部分
3. 替换 `your-api-key-here` 为实际的 API Key

### Q: 端口被占用

**A**: SDK 默认使用 8080 端口,如果被占用,可以修改配置或停止占用端口的程序。

## 自定义配置

所有示例都支持自定义配置,修改 `sdk.Config`:

```go
client, err := sdk.New(&sdk.Config{
    DatabasePath:  "./custom/path/db.db",  // 自定义数据库路径
    EnableBrowser: true,
    EnableScript:  true,
    EnableAgent:   false,
    LogLevel:      "debug",                 // 日志级别: debug/info/warn/error
    LogOutput:     "./logs/app.log",        // 日志输出文件
})
```

## 示例输出

### Basic Example 输出示例:

```
Starting browser...
✓ Browser manager initialized successfully
Opening page...
Creating script...
✓ Script created with ID: abc123...

Listing all scripts...
Found 1 scripts:
  - Example Script: A simple example script

Playing script...
✓ Script execution result:
  Status: success
  Duration: 1234 ms
  Extracted data:
    title: Example Domain

✓ Example completed successfully!
```

### Advanced Example 输出示例:

```
Starting browser...

=== Scenario 1: Data extraction ===
✓ Created script: def456...
Extraction result:
  heading: Example Domain
  paragraph: This domain is for use in...

=== Scenario 2: Batch page visits ===
Visiting page 1: https://www.example.com
  ✓ Status: success, Duration: 890ms
...

✓ Advanced example completed!
```

## 下一步

学习完示例后,你可以:

1. 查看 [USAGE.md](../USAGE.md) 了解完整 API
2. 查看 [DESIGN.md](../DESIGN.md) 了解架构设计
3. 在自己的项目中集成 SDK

## 获取帮助

如有问题,请:
1. 检查日志输出
2. 查看文档
3. 提交 Issue
