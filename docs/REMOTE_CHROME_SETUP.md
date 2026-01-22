# 远程 Chrome 配置指南

本指南说明如何配置 BrowserPilot 连接到远程 Chrome 浏览器。

## 📖 相关文档

- **Docker 部署**: 查看 [docker/chrome/QUICKSTART.md](docker/chrome/QUICKSTART.md) 了解如何使用 Docker 快速启动 Chrome
- **完整 Docker 文档**: [docker/chrome/README.md](docker/chrome/README.md)

## 功能特点

- ✅ 支持连接本地或远程 Chrome
- ✅ 优先使用配置的远程 URL
- ✅ 未配置远程 URL 时自动启动本地 Chrome
- ✅ 安全断开连接，不会关闭远程 Chrome

## 配置方法

### 1. 编辑配置文件

编辑 `backend/config.toml`，在 `[browser]` 部分添加 `control_url`：

```toml
[browser]
# 远程 Chrome URL（留空则启动本地浏览器）
control_url = "http://192.168.1.100:9222"
```

### 2. URL 格式

支持以下格式：

- **HTTP**: `http://localhost:9222`
- **WebSocket**: `ws://localhost:9222`
- **远程服务器**: `http://192.168.1.100:9222`
- **Docker 容器**: `http://chrome-container:9222`

## 使用场景

### 场景 1: 连接本地已运行的 Chrome

1. 启动 Chrome 并开启远程调试：

```bash
# Windows
"C:\Program Files\Google\Chrome\Application\chrome.exe" --remote-debugging-port=9222 --no-first-run

# Linux
google-chrome --remote-debugging-port=9222 --no-first-run

# macOS
"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --remote-debugging-port=9222 --no-first-run
```

2. 配置文件设置：

```toml
[browser]
control_url = "http://localhost:9222"
```

### 场景 2: 连接远程服务器的 Chrome

1. 在远程服务器上启动 Chrome：

```bash
google-chrome --remote-debugging-port=9222 --remote-debugging-address=0.0.0.0 --no-first-run
```

2. 配置文件设置：

```toml
[browser]
control_url = "http://192.168.1.100:9222"  # 替换为实际 IP
```

### 场景 3: Docker 容器中的 Chrome

🐳 **推荐使用 Docker 部署！** 详细文档请参考:
- [快速开始指南](docker/chrome/QUICKSTART.md) - 5 分钟快速上手
- [完整文档](docker/chrome/README.md) - 包含 Dockerfile、docker-compose 和故障排查

1. 使用 Docker 运行 Chrome（快速方式）：

```bash
# 使用官方镜像（推荐）
docker run -d \
  --name browserpilot-chrome \
  -p 9222:9222 \
  --shm-size=2g \
  zenika/alpine-chrome:latest \
  --no-sandbox \
  --disable-dev-shm-usage \
  --remote-debugging-address=0.0.0.0 \
  --remote-debugging-port=9222

# 或使用我们提供的启动脚本
cd docker/chrome
./start-chrome.sh
```

2. 配置文件设置：

```toml
[browser]
control_url = "http://localhost:9222"
```

### 场景 4: 本地模式（默认）

如果不配置 `control_url` 或设置为空字符串，将自动启动本地 Chrome：

```toml
[browser]
control_url = ""  # 或者删除这一行
bin_path = "C:/Program Files/Google/Chrome/Application/chrome.exe"
user_data_dir = "./chrome_user_data"
```

## 日志输出

### 远程模式

```
[INFO] Using remote Chrome browser
[INFO] Control URL: http://192.168.1.100:9222
[INFO] Disconnected from remote browser successfully
```

### 本地模式

```
[INFO] Starting local Chrome browser...
[INFO] Starting browser process...
[INFO] Browser control URL: ws://127.0.0.1:xxxxx
[INFO] Browser process terminated
[INFO] Browser fully closed, user data saved
```

## 注意事项

1. **远程模式下被忽略的配置**：
   - `bin_path` - 浏览器路径
   - `user_data_dir` - 用户数据目录
   - `headless` - 无头模式
   - 这些配置仅在本地模式下生效

2. **停止浏览器行为**：
   - 远程模式：仅断开连接，不关闭浏览器进程
   - 本地模式：关闭所有页面并终止浏览器进程

3. **安全建议**：
   - 远程调试端口不要暴露到公网
   - 使用防火墙限制访问来源
   - 建议在受信任的内网环境使用

4. **网络要求**：
   - 确保网络可达性
   - 检查防火墙规则
   - 验证端口未被占用

## 故障排查

### 连接失败

如果连接远程 Chrome 失败，请检查：

1. Chrome 是否正在运行
2. 远程调试端口是否正确
3. 网络连接是否正常
4. 防火墙是否阻止连接

### 测试连接

在浏览器中访问：`http://localhost:9222/json/version`

应该返回类似以下的 JSON：

```json
{
  "Browser": "Chrome/120.0.6099.109",
  "Protocol-Version": "1.3",
  "User-Agent": "Mozilla/5.0...",
  "WebKit-Version": "537.36"
}
```

## 示例配置

### 完整的远程配置示例

```toml
[server]
host = "0.0.0.0"
port = "8080"

[browser]
# 使用远程 Chrome
control_url = "http://192.168.1.100:9222"

# 以下配置在远程模式下会被忽略
bin_path = ""
user_data_dir = ""
```

### 完整的本地配置示例

```toml
[server]
host = "0.0.0.0"
port = "8080"

[browser]
# 本地模式（不配置 control_url）
control_url = ""
bin_path = "C:/Program Files/Google/Chrome/Application/chrome.exe"
user_data_dir = "./chrome_user_data"
```

