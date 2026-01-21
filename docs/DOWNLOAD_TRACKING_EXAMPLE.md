# 下载文件追踪功能 - 代码示例

## 实现对比

### 旧方案（文件系统扫描）❌

```go
// 问题：需要扫描文件系统，无法准确匹配并发下载
func (p *Player) TrackDownloadsEnd(ctx context.Context, initialFiles map[string]time.Time) {
    entries, err := os.ReadDir(p.downloadPath)
    // 遍历所有文件，根据时间戳猜测哪些是新下载的
    for _, entry := range entries {
        // 可能误判其他脚本的下载
    }
}
```

### 新方案（事件驱动）✅

```go
// 优势：直接从浏览器事件获取文件名，精确匹配
func (p *Player) StartDownloadListener(ctx context.Context, browser *rod.Browser) {
    downloadMap := make(map[string]string) // GUID -> FileName
    
    // 1. 监听下载开始：获取文件名
    go browser.Context(p.downloadCtx).EachEvent(func(e *proto.BrowserDownloadWillBegin) {
        downloadMap[e.GUID] = e.SuggestedFilename
        logger.Info(ctx, "📥 Download will begin: %s (GUID: %s)", e.SuggestedFilename, e.GUID)
    })()
    
    // 2. 监听下载完成：通过 GUID 获取文件名
    go browser.Context(p.downloadCtx).EachEvent(func(e *proto.BrowserDownloadProgress) {
        if e.State == proto.BrowserDownloadProgressStateCompleted {
            fileName := downloadMap[e.GUID]  // 精确获取文件名
            fullPath := filepath.Join(p.downloadPath, fileName)
            p.downloadedFiles = append(p.downloadedFiles, fullPath)
        }
    })()
}
```

## 事件流程

### 单个文件下载

```
时间线：0ms ───────────────────────> 2000ms
         │                              │
         ├── DownloadWillBegin          ├── DownloadProgress.Completed
         │   GUID: abc-123              │   GUID: abc-123
         │   FileName: report.pdf       │   TotalBytes: 2500000
         │                              │
         ├── 映射表[abc-123] = "report.pdf"
         │                              │
         │                              ├── 从映射表获取: "report.pdf"
         │                              ├── 构建路径: /downloads/report.pdf
         │                              └── 添加到结果列表
```

### 并发下载（多个脚本同时执行）

```
脚本 A:
  DownloadWillBegin(GUID: abc-123, FileName: file1.pdf)
    ↓
  映射表A[abc-123] = "file1.pdf"
    ↓
  DownloadProgress.Completed(GUID: abc-123)
    ↓
  从映射表A获取 → file1.pdf ✅

脚本 B (同时进行):
  DownloadWillBegin(GUID: xyz-789, FileName: file2.pdf)
    ↓
  映射表B[xyz-789] = "file2.pdf"
    ↓
  DownloadProgress.Completed(GUID: xyz-789)
    ↓
  从映射表B获取 → file2.pdf ✅

结果：
- 脚本 A 只获取 file1.pdf
- 脚本 B 只获取 file2.pdf
- 完全隔离，互不干扰 ✅
```

## 浏览器自动重命名处理

### 场景：下载同名文件

```go
// 第一次下载
DownloadWillBegin(GUID: aaa, FileName: "document.pdf")
DownloadProgress.Completed(GUID: aaa)
// ✓ 保存为: /downloads/document.pdf

// 第二次下载同名文件
DownloadWillBegin(GUID: bbb, FileName: "document.pdf")
DownloadProgress.Completed(GUID: bbb)
// 检查文件存在性
if _, err := os.Stat("/downloads/document.pdf"); os.IsNotExist(err) {
    // 文件不存在，被浏览器重命名了
    actualFile := p.findSimilarFile("document.pdf")
    // 找到: "document (1).pdf"
    logger.Info("File was renamed: document.pdf -> document (1).pdf")
}
// ✓ 保存为: /downloads/document (1).pdf
```

### findSimilarFile 实现

```go
func (p *Player) findSimilarFile(originalName string) string {
    // 输入: "document.pdf"
    ext := ".pdf"                    // 扩展名
    nameWithoutExt := "document"     // 文件名（无扩展名）
    
    // 扫描下载目录
    for _, entry := range entries {
        name := entry.Name()
        
        // 匹配模式：
        // ✓ document.pdf
        // ✓ document (1).pdf
        // ✓ document (2).pdf
        // ✗ document2.pdf (不匹配)
        // ✗ mydocument.pdf (不匹配)
        
        if strings.HasPrefix(name, nameWithoutExt) && 
           strings.HasSuffix(name, ext) {
            // 检查中间是否是 " (数字)" 格式
            if name == originalName || 
               (name[len(nameWithoutExt)] == ' ' && 
                name[len(nameWithoutExt)+1] == '(') {
                return name
            }
        }
    }
    
    return ""
}
```

## 完整示例

### 测试脚本

```javascript
// 脚本内容：下载两个文件
1. 访问 https://example.com/downloads
2. 点击 "Download Report"  → report.pdf
3. 等待 2 秒
4. 点击 "Download Invoice" → invoice.xlsx
5. 等待 2 秒
```

### 执行日志

```
[INFO] Download tracking enabled for playback, path: /root/code/browserpilot/downloads
[INFO] Starting download event listener for path: /root/code/browserpilot/downloads
[INFO] Download event listener started

[INFO] 执行步骤 1: 访问 https://example.com/downloads
[INFO] 执行步骤 2: 点击 "Download Report"
[INFO] 📥 Download will begin: report.pdf (GUID: 12345-abc)

[INFO] 执行步骤 3: 等待 2 秒
[INFO] ✓ Download completed: /root/code/browserpilot/downloads/report.pdf (2.35 MB, GUID: 12345-abc)

[INFO] 执行步骤 4: 点击 "Download Invoice"
[INFO] 📥 Download will begin: invoice.xlsx (GUID: 67890-xyz)

[INFO] 执行步骤 5: 等待 2 秒
[INFO] ✓ Download completed: /root/code/browserpilot/downloads/invoice.xlsx (0.85 MB, GUID: 67890-xyz)

[INFO] Download event listener stopped
[INFO] ✓ Total downloaded files: 2
[INFO]   #1: /root/code/browserpilot/downloads/report.pdf
[INFO]   #2: /root/code/browserpilot/downloads/invoice.xlsx
```

### API 返回

```json
{
  "success": true,
  "message": "Script replay completed",
  "extracted_data": {
    "downloaded_files": [
      "/root/code/browserpilot/downloads/report.pdf",
      "/root/code/browserpilot/downloads/invoice.xlsx"
    ]
  }
}
```

## 关键优势总结

| 特性 | 旧方案（文件系统） | 新方案（事件驱动） |
|------|-------------------|-------------------|
| **文件名获取** | ❌ 扫描文件系统猜测 | ✅ 直接从事件获取 |
| **并发安全** | ❌ 可能误判 | ✅ GUID 精确匹配 |
| **性能** | ❌ 需要扫描目录 | ✅ 无需扫描 |
| **准确性** | ❌ 基于时间戳推测 | ✅ 100% 精确 |
| **重命名处理** | ❌ 可能遗漏 | ✅ 智能查找 |
| **文件大小** | ❌ 需要额外读取 | ✅ 事件直接提供 |

## 参考

- Recorder 实现：`backend/services/browser/recorder.go:1037-1100`
- Player 实现：`backend/services/browser/player.go`
- 完整文档：`docs/DOWNLOAD_TRACKING.md`
