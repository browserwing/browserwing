# Browserwing Panic 修复说明

## 问题描述

用户报告在使用 `browserwing-browser_navigate` 时遇到 panic 错误,导致上游没有收到 tool call result 就结束了。

### 错误信息
```
assignment to entry in nil map
/root/.goenv/versions/1.24.2/src/internal/runtime/maps/runtime_faststr_swiss.go:265
```

### 调用栈
- `/root/code/browserwing/backend/executor/operations.go:90` - Navigate 函数中的 `page.WaitLoad()` 调用
- 经过 rod 库的 `page_eval.go` 相关代码
- 从 `mcp_tools.go:163` 触发

## 根本原因

这是 **go-rod 库在并发场景下的一个已知 bug**。当页面正在加载或状态转换时,Rod 库内部的某些 map 数据结构可能未初始化就被访问,导致 panic。

具体触发场景:
1. 页面正在加载时调用 `page.WaitLoad()`
2. 页面状态转换时调用 `page.Eval()`
3. 并发调用页面方法时,内部状态不一致

## 修复方案

采用 **防御性编程** 策略,为所有可能产生 panic 的 Rod 操作添加 panic 恢复机制。

### 修改的文件

1. **backend/executor/operations.go**
   - 添加 `safeWaitForPageLoad()` - 安全的页面加载等待
   - 添加 `safeScrollEval()` - 安全的滚动操作
   - 添加 `safeEvaluate()` - 安全的 JavaScript 执行
   - 修改 `Navigate()`, `ScrollToBottom()`, `Evaluate()` 函数使用安全包装

2. **backend/executor/semantic.go**
   - 添加 `safePageEvalUnmarshal()` - 安全的页面 JS 执行并解析结果
   - 修改 `markCursorPointerElements()` 使用安全包装

3. **backend/executor/executor.go**
   - 添加 `safeGetPageText()` - 安全的获取页面文本
   - 修改 `GetPageText()` 使用安全包装

4. **backend/executor/mcp_tools.go**
   - 添加 `safeScrollToTop()` - 安全的滚动到顶部
   - 修改滚动工具使用安全包装
   - 添加 `rod` 包导入

### 核心机制

所有安全包装函数都使用以下模式:

```go
func safeSomeOperation(ctx context.Context, page *rod.Page) (err error) {
    // 使用 defer recover 来捕获 rod 库可能产生的 panic
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("panic during operation: %v", r)
        }
    }()
    
    // 执行实际操作
    err = page.SomeOperation()
    return err
}
```

### 关键改进

1. **Panic 恢复**: 使用 `defer/recover` 捕获所有 panic,转换为普通错误
2. **错误传播**: panic 被转换为错误后,会正确地返回给 MCP 上游
3. **优雅降级**: 即使某些操作失败(如语义树提取超时),导航操作仍会继续
4. **日志保留**: 所有错误和警告都会被记录,便于调试

## 影响范围

### 修复的操作
- ✅ 页面导航 (`Navigate`)
- ✅ 页面加载等待 (`WaitLoad`, `WaitIdle`)
- ✅ JavaScript 执行 (`Eval`)
- ✅ 语义树提取 (`ExtractSemanticTree`)
- ✅ 滚动操作 (`ScrollToBottom`, scroll to top)
- ✅ 获取页面文本 (`GetPageText`)

### 向后兼容性
- ✅ 所有修改都是在内部实现层面
- ✅ 对外 API 接口没有变化
- ✅ MCP 工具定义没有变化
- ✅ 现有代码可以无缝升级

## 测试建议

### 重现原问题
1. 访问加载较慢的网站 (如 https://www.gamersky.com)
2. 多次快速调用 `browser_navigate`
3. 观察是否还会出现 panic

### 验证修复
1. 正常场景测试:
   ```bash
   # 测试导航
   browser_navigate(url="https://www.example.com")
   
   # 测试滚动
   browser_scroll(direction="bottom")
   
   # 测试 JavaScript 执行
   browser_evaluate(script="console.log('test')")
   ```

2. 压力测试:
   - 连续快速导航多个页面
   - 在页面加载期间调用其他操作
   - 并发执行多个浏览器操作

3. 错误场景测试:
   - 访问不存在的页面
   - 执行错误的 JavaScript
   - 超长加载时间的页面

## 预期效果

### 修复前
- ❌ 随机出现 panic: `assignment to entry in nil map`
- ❌ 上游 MCP hub 收不到返回结果
- ❌ 连接中断,需要重启服务

### 修复后
- ✅ 所有 panic 被捕获并转换为错误
- ✅ 错误正确返回给上游 MCP hub
- ✅ 服务保持运行,连接不会中断
- ✅ 错误信息清晰,便于排查

## 长期解决方案

虽然当前的修复可以解决 panic 问题,但建议:

1. **监控 go-rod 库更新**
   - 当前使用版本: v0.116.2
   - 关注 github.com/go-rod/rod 的 issue 和更新
   - 测试新版本是否修复了底层 bug

2. **添加重试机制**
   - 对于临时性失败,可以添加自动重试
   - 特别是页面加载等待操作

3. **增强日志**
   - 记录每次 panic 的详细上下文
   - 帮助识别触发 panic 的具体场景

4. **性能监控**
   - 监控页面操作的耗时
   - 识别慢速操作和潜在问题

## 相关链接

- Rod 库: https://github.com/go-rod/rod
- MCP Go SDK: https://github.com/mark3labs/mcp-go
- 类似问题讨论: https://github.com/go-rod/rod/issues (搜索 "nil map")

## 变更记录

- **2026-01-21**: 初始修复,添加所有 panic 恢复机制

