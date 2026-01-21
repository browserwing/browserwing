# 下载文件追踪功能

## 功能概述

在脚本回放过程中，系统会自动监听并追踪浏览器的下载行为，将下载的文件路径作为提取数据返回。

## 工作原理

### 1. 启用下载事件

在浏览器启动时：
- 设置 `BrowserSetDownloadBehavior` 的 `EventsEnabled: true`
- 浏览器会发送 `BrowserDownloadProgress` 事件
- 配置下载路径到 `downloads/` 目录

### 2. 启动事件监听

在脚本回放开始前：
- 创建下载事件监听器
- 订阅 `BrowserDownloadWillBegin` 事件（获取文件名）
- 订阅 `BrowserDownloadProgress` 事件（获取下载状态）
- 创建 GUID 到文件名的映射表

### 3. 监听下载事件

在脚本执行过程中：
- **`BrowserDownloadWillBegin`**：记录 GUID 和建议的文件名
- **`BrowserDownloadProgress.Completed`**：通过 GUID 获取文件名，构建完整路径
- **`BrowserDownloadProgress.Canceled`**：清理取消的下载记录

### 4. 文件名处理

下载完成后：
- 从 GUID 映射表中获取原始文件名
- 构建完整的文件路径
- 检查文件是否存在（处理浏览器自动重命名）
- 如果文件被重命名（如 `file.pdf` → `file (1).pdf`），自动查找实际文件名

### 5. 停止监听并返回

脚本回放结束后：
- 停止下载事件监听
- 收集所有下载的文件路径
- 添加到 `ExtractedData` 中返回

```json
{
  "extracted_field_1": "value1",
  "extracted_field_2": "value2",
  "downloaded_files": [
    "/path/to/downloads/document.pdf",
    "/path/to/downloads/report.xlsx"
  ]
}
```

## 使用示例

### API 调用

```bash
# 回放脚本
curl -X POST http://localhost:8080/api/v1/scripts/{script_id}/play

# 响应示例
{
  "success": true,
  "message": "Script replay completed",
  "extracted_data": {
    "product_title": "MacBook Pro",
    "price": "$1999",
    "downloaded_files": [
      "/root/code/browserpilot/downloads/product_spec.pdf",
      "/root/code/browserpilot/downloads/warranty_info.pdf"
    ]
  }
}
```

### 场景示例

#### 场景 1：下载单个文件

**脚本步骤：**
1. 访问文档下载页面
2. 点击"下载 PDF"按钮
3. 等待下载完成

**返回数据：**
```json
{
  "downloaded_files": [
    "/root/code/browserpilot/downloads/document_2025.pdf"
  ]
}
```

#### 场景 2：批量下载

**脚本步骤：**
1. 访问报表页面
2. 选择多个报表
3. 点击"批量下载"
4. 等待所有文件下载完成

**返回数据：**
```json
{
  "downloaded_files": [
    "/root/code/browserpilot/downloads/sales_report_jan.xlsx",
    "/root/code/browserpilot/downloads/sales_report_feb.xlsx",
    "/root/code/browserpilot/downloads/sales_report_mar.xlsx"
  ]
}
```

#### 场景 3：下载 + 数据抓取

**脚本步骤：**
1. 访问产品页面
2. 抓取产品信息（价格、标题）
3. 下载产品说明书
4. 下载技术规格表

**返回数据：**
```json
{
  "product_title": "iPhone 15 Pro",
  "price": "$999",
  "stock_status": "In Stock",
  "downloaded_files": [
    "/root/code/browserpilot/downloads/iphone15_manual.pdf",
    "/root/code/browserpilot/downloads/iphone15_tech_specs.pdf"
  ]
}
```

## 实现细节

### Player 结构体新增字段

```go
type Player struct {
    // ... 其他字段
    downloadedFiles  []string            // 下载的文件路径列表
    downloadPath     string              // 下载目录路径
    downloadCtx      context.Context     // 下载监听上下文
    downloadCancel   context.CancelFunc  // 取消下载监听
}
```

### 核心方法

#### StartDownloadListener

```go
func (p *Player) StartDownloadListener(ctx context.Context, browser *rod.Browser)
```

- **作用**：启动下载事件监听
- **参数**：上下文和浏览器实例
- **调用时机**：脚本回放开始前
- **功能**：
  - 创建 GUID 到文件名的映射表
  - 订阅 `BrowserDownloadWillBegin` 事件（获取建议的文件名）
  - 订阅 `BrowserDownloadProgress` 事件（监听下载状态）
  - 下载完成时通过 GUID 查找文件名并构建完整路径
  - 处理浏览器自动重命名文件的情况（如 `file.pdf` → `file (1).pdf`）

#### findSimilarFile

```go
func (p *Player) findSimilarFile(originalName string) string
```

- **作用**：查找相似的文件名
- **参数**：原始文件名
- **返回**：实际存在的文件名
- **用途**：处理浏览器自动重命名文件的情况

#### StopDownloadListener

```go
func (p *Player) StopDownloadListener(ctx context.Context)
```

- **作用**：停止下载事件监听
- **调用时机**：脚本回放结束后
- **功能**：
  - 取消事件监听上下文
  - 记录最终下载的文件统计

#### GetDownloadedFiles

```go
func (p *Player) GetDownloadedFiles() []string
```

- **作用**：获取下载的文件列表
- **返回**：文件完整路径数组

## 配置说明

### 下载目录

默认下载目录：`./downloads`

可以通过环境变量或配置修改：
```bash
# 在浏览器启动时设置
export DOWNLOAD_PATH=/custom/path/to/downloads
```

### 权限要求

- 下载目录必须存在且可写
- 系统会自动创建不存在的下载目录
- 确保进程有读写权限

## 日志输出

### 成功场景

```
[INFO] Download tracking enabled for playback, path: /root/code/browserpilot/downloads
[INFO] Starting download event listener for path: /root/code/browserpilot/downloads
[INFO] Download event listener started
[INFO] 📥 Download will begin: report.pdf (GUID: 12345-abcde-67890)
[INFO] ✓ Download completed: /root/code/browserpilot/downloads/report.pdf (2.35 MB, GUID: 12345-abcde-67890)
[INFO] 📥 Download will begin: invoice.xlsx (GUID: 98765-fghij-43210)
[INFO] ✓ Download completed: /root/code/browserpilot/downloads/invoice.xlsx (0.85 MB, GUID: 98765-fghij-43210)
[INFO] Download event listener stopped
[INFO] ✓ Total downloaded files: 2
[INFO]   #1: /root/code/browserpilot/downloads/report.pdf
[INFO]   #2: /root/code/browserpilot/downloads/invoice.xlsx
[INFO] [PlayScript] Downloaded files count: 2
[INFO] [PlayScript] Downloaded file #1: /root/code/browserpilot/downloads/report.pdf
[INFO] [PlayScript] Downloaded file #2: /root/code/browserpilot/downloads/invoice.xlsx
```

### 浏览器自动重命名文件

```
[INFO] 📥 Download will begin: document.pdf (GUID: abc-123)
[INFO] File was renamed by browser: document.pdf -> document (1).pdf
[INFO] ✓ Download completed: /root/code/browserpilot/downloads/document (1).pdf (1.20 MB, GUID: abc-123)
```

### 无下载场景

```
[INFO] Download tracking enabled for playback, path: /root/code/browserpilot/downloads
[INFO] Starting download event listener for path: /root/code/browserpilot/downloads
[INFO] Download event listener started
[INFO] Download event listener stopped
[INFO] No files downloaded during script execution
```

## 注意事项

### 1. 文件名冲突

如果下载目录中已存在同名文件：
- 浏览器通常会自动重命名（添加数字后缀）
- 例如：`document.pdf` → `document (1).pdf`
- 系统会正确识别新下载的文件

### 2. 下载时间

- 系统使用事件实时监听下载进度
- 如果下载很慢，建议在脚本末尾添加等待步骤
- 可以添加 `wait` 动作等待下载完成（建议至少等待 1-2 秒）

### 3. 并发执行

- ✅ **现在支持并发安全**
- 通过浏览器下载事件的 GUID 和时间戳精确匹配文件
- 每个脚本只会追踪自己触发的下载
- 不会误获取其他并发脚本的下载文件

### 4. 文件清理

- 系统不会自动清理下载的文件
- 建议定期清理下载目录
- 可以通过脚本或定时任务实现

### 5. EventsEnabled 配置

- **必须在浏览器启动时设置 `EventsEnabled: true`**
- 这已经在 `manager.go` 中默认启用
- 如果未启用，下载事件将不会被触发

## 最佳实践

### 1. 等待下载完成

在脚本末尾添加等待：

```json
{
  "type": "wait",
  "value": "3000",
  "comment": "等待文件下载完成"
}
```

### 2. 验证文件存在

在代码中验证下载的文件：

```go
if downloadedFiles, ok := result.ExtractedData["downloaded_files"].([]string); ok {
    for _, file := range downloadedFiles {
        if _, err := os.Stat(file); err == nil {
            fmt.Printf("✓ File exists: %s\n", file)
        } else {
            fmt.Printf("✗ File not found: %s\n", file)
        }
    }
}
```

### 3. 移动文件到目标位置

```go
for _, file := range downloadedFiles {
    dest := filepath.Join("/target/path", filepath.Base(file))
    if err := os.Rename(file, dest); err != nil {
        log.Printf("Failed to move file: %v", err)
    }
}
```

## 故障排查

### 问题：未检测到下载的文件

**可能原因：**
1. 下载时间过长，脚本已结束
2. 浏览器下载被阻止（弹出窗口/权限）
3. 下载目录路径不正确

**解决方法：**
1. 在脚本末尾添加等待时间
2. 检查浏览器配置和权限
3. 查看日志确认下载路径

### 问题：文件路径不正确

**可能原因：**
- 相对路径 vs 绝对路径

**解决方法：**
- 系统返回的都是绝对路径
- 使用 `filepath.Abs()` 转换路径

### 问题：文件已存在被覆盖

**可能原因：**
- 浏览器默认行为

**解决方法：**
- 使用时间戳命名
- 预先清空下载目录

## 相关代码

- `backend/services/browser/player.go` - Player 实现
- `backend/services/browser/manager.go` - PlayScript 方法
- `backend/models/script_execution.go` - 数据模型

## 更新日志

- **2025-01-18**: 初始实现下载文件追踪功能
  - 使用 `BrowserDownloadWillBegin` 事件直接获取文件名
  - 使用 `BrowserDownloadProgress` 事件监听下载状态
  - 支持并发安全，每个脚本只追踪自己的下载
  - 通过 GUID 精确匹配下载的文件，无需扫描文件系统
  - 自动处理浏览器重命名文件的情况
  - 显示文件大小信息
  - 将下载文件路径添加到提取数据中返回
