# 浏览器登录状态保持指南

## 问题说明

如果你发现每次重新打开浏览器后登录状态丢失，需要重新登录，这是因为浏览器的用户数据没有被正确保存。

## 解决方案

### 1. 配置用户数据目录

在 `config.toml` 中正确配置 `user_data_dir`：

```toml
[browser]
headless = false
user_data_dir = "C:/Users/YourName/AppData/Local/browserwing/chrome_data"
```

**关键点：**
- ✅ **必须使用绝对路径**
- ✅ **Windows 路径使用正斜杠 `/` 而不是反斜杠 `\`**
- ✅ **确保该目录有读写权限**
- ✅ **该目录会存储 Cookie、缓存、登录状态等数据**

### 2. 推荐的目录位置

#### Windows
```toml
# 方案1：使用用户 AppData 目录（推荐）
user_data_dir = "C:/Users/Administrator/AppData/Local/browserwing/chrome_data"

# 方案2：使用项目目录
user_data_dir = "C:/path/to/your/project/chrome_user_data"

# 方案3：使用桌面目录
user_data_dir = "C:/Users/Administrator/Desktop/chrome_data"
```

#### Linux/Mac
```toml
# 方案1：使用项目相对路径
user_data_dir = "./chrome_user_data"

# 方案2：使用用户目录
user_data_dir = "/home/username/.browserwing/chrome_data"
```

### 3. 验证配置是否生效

启动浏览器后，检查日志输出：

```
正在启动浏览器...
使用浏览器路径: C:/Program Files/Google/Chrome/Application/chrome.exe
使用用户数据目录: C:/Users/Administrator/AppData/Local/browserwing/chrome_data
浏览器启动成功
```

如果看到 "使用用户数据目录" 的日志，说明配置已生效。

### 4. 检查目录是否创建成功

启动浏览器后，检查配置的目录是否被创建，里面应该包含：

```
chrome_user_data/
├── Default/
│   ├── Cookies
│   ├── Local Storage/
│   ├── Cache/
│   └── ...
├── First Run
└── ...
```

### 5. 常见问题

#### 问题1：目录权限不足
**现象**：日志显示"创建用户数据目录失败"

**解决**：
- 确保运行程序的用户对该目录有读写权限
- 尝试更换到有权限的目录（如用户主目录下）

#### 问题2：路径格式错误
**现象**：浏览器启动失败或没有使用配置的目录

**解决**：
- Windows 路径使用 `/` 而不是 `\`
- 使用绝对路径而不是相对路径
- 路径中不要有中文或特殊字符

#### 问题3：多个浏览器实例冲突
**现象**：提示"浏览器配置文件正在使用中"

**解决**：
- 关闭所有 Chrome 浏览器窗口
- 或者为不同的自动化任务使用不同的 `user_data_dir`

#### 问题4：登录状态仍然丢失
**可能原因**：
1. 网站使用了额外的安全验证（如设备指纹）
2. Cookie 有时效性限制
3. 浏览器标识被识别为自动化工具

**解决**：
- 确保配置中设置了 `disable-blink-features: AutomationControlled`
- 尝试在正常 Chrome 浏览器中登录并复制 Cookie
- 某些网站可能需要额外的验证步骤

### 6. 高级配置

如果需要更隐蔽的自动化，可以添加更多启动参数：

```go
// 在 manager.go 中添加
l := launcher.New().
    Headless(false).
    Devtools(false).
    Set("disable-blink-features", "AutomationControlled").
    Set("disable-infobars").
    Delete("enable-automation").
    Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
```

### 7. 测试登录状态保持

**步骤**：
1. 启动浏览器
2. 打开小红书/微信公众号等网站
3. 完成登录
4. 关闭浏览器
5. 重新启动浏览器
6. 再次打开相同网站
7. **预期结果**：应该自动保持登录状态，无需重新登录

### 8. 调试技巧

如果仍然有问题，可以：

1. **检查日志级别**：设置为 `debug` 查看详细信息
```toml
[log]
level = "debug"
```

2. **手动检查 Cookie**：
   - 在 `user_data_dir/Default/Cookies` 查看是否保存了网站的 Cookie
   - 使用 Chrome 的 SQLite 工具查看 Cookie 数据库

3. **对比正常浏览器**：
   - 使用正常 Chrome 指定 `--user-data-dir` 参数启动
   - 对比两者的行为差异

## 总结

保持登录状态的**核心要点**：

✅ **正确配置 `user_data_dir`**  
✅ **使用绝对路径**  
✅ **确保目录权限**  
✅ **关闭无头模式（headless = false）**  
✅ **隐藏自动化特征**  

配置正确后，浏览器会像普通 Chrome 一样保存所有数据，登录状态将会持久化。
