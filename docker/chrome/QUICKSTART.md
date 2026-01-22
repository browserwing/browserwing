# Chrome Docker 快速开始

5 分钟内启动 Docker Chrome 并连接到 BrowserPilot！

## 🚀 最快方式（推荐）

### 一键启动

```bash
cd docker/chrome
./start-chrome.sh
```

选择选项 1（使用官方镜像），然后等待启动完成。

### 手动启动（如果脚本不工作）

```bash
docker run -d \
  --name browserpilot-chrome \
  -p 9222:9222 \
  --shm-size=2g \
  zenika/alpine-chrome:latest \
  --no-sandbox \
  --disable-dev-shm-usage \
  --remote-debugging-address=0.0.0.0 \
  --remote-debugging-port=9222
```

## ✅ 验证安装

### 1. 检查容器状态

```bash
docker ps | grep chrome
```

应该看到容器正在运行。

### 2. 测试连接

```bash
curl http://localhost:9222/json/version
```

应该返回 JSON 格式的版本信息。

### 3. 运行测试脚本（可选）

```bash
./test-chrome.sh
```

## 🔧 配置 BrowserPilot

编辑 `backend/config.toml`：

```toml
[browser]
control_url = "http://localhost:9222"
```

**就这么简单！** 现在启动 BrowserPilot，它会自动连接到 Docker Chrome。

## 📦 三种方案对比

| 方案 | 命令 | 优点 | 缺点 |
|-----|------|------|------|
| **Zenika 官方镜像** | `./start-chrome.sh` 选项 1 | ✅ 最快<br>✅ 维护良好<br>✅ 500MB | ⚠️ Chromium 不是 Chrome |
| **Alpine Chromium** | `./start-chrome.sh` 选项 3 | ✅ 最轻量 200MB<br>✅ 可自定义 | ⚠️ 需要构建时间 |
| **Debian Chrome** | `./start-chrome.sh` 选项 4 | ✅ 完整 Chrome<br>✅ 功能最全 | ❌ 1GB 大小<br>❌ 构建慢 |

## 🎯 完整示例

### 步骤 1: 启动 Chrome 容器

```bash
docker run -d \
  --name browserpilot-chrome \
  -p 9222:9222 \
  --shm-size=2g \
  --restart unless-stopped \
  zenika/alpine-chrome:latest \
  --no-sandbox \
  --disable-dev-shm-usage \
  --remote-debugging-address=0.0.0.0 \
  --remote-debugging-port=9222
```

### 步骤 2: 获取 Control URL

```bash
# 方法 1: 本地访问
echo "http://localhost:9222"

# 方法 2: 通过容器 IP
docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' browserpilot-chrome
# 示例输出: 172.17.0.2
# Control URL: http://172.17.0.2:9222

# 方法 3: 远程访问（替换为你的主机 IP）
hostname -I | awk '{print $1}'
# 示例输出: 192.168.1.100
# Control URL: http://192.168.1.100:9222
```

### 步骤 3: 验证连接

```bash
curl http://localhost:9222/json/version
```

期望输出：
```json
{
  "Browser": "Chrome/120.0.6099.109",
  "Protocol-Version": "1.3",
  "User-Agent": "Mozilla/5.0 ...",
  "webSocketDebuggerUrl": "ws://localhost:9222/devtools/browser/..."
}
```

### 步骤 4: 配置 BrowserPilot

编辑 `backend/config.toml`：

```toml
[server]
host = "0.0.0.0"
port = "8080"

[browser]
# 使用 Docker Chrome
control_url = "http://localhost:9222"

# 以下配置在远程模式下会被忽略
# bin_path = ""
# user_data_dir = ""
```

### 步骤 5: 启动 BrowserPilot

```bash
cd backend
go run main.go
```

或者使用已编译的二进制：

```bash
./browserpilot
```

查看日志，应该看到：

```
[INFO] Using remote Chrome browser
[INFO] Control URL: http://localhost:9222
[INFO] Browser started successfully
```

## 🐳 使用 docker-compose（推荐生产环境）

创建 `docker-compose.yml`：

```yaml
version: '3.8'

services:
  chrome:
    image: zenika/alpine-chrome:latest
    container_name: browserpilot-chrome
    shm_size: 2gb
    ports:
      - "9222:9222"
    command:
      - --no-sandbox
      - --disable-dev-shm-usage
      - --remote-debugging-address=0.0.0.0
      - --remote-debugging-port=9222
    restart: unless-stopped
    networks:
      - browserpilot-net

  backend:
    build: ./backend
    container_name: browserpilot-backend
    ports:
      - "8080:8080"
    environment:
      # 使用服务名作为主机名
      - BROWSER_CONTROL_URL=http://chrome:9222
    depends_on:
      - chrome
    restart: unless-stopped
    networks:
      - browserpilot-net

networks:
  browserpilot-net:
    driver: bridge
```

启动所有服务：

```bash
docker-compose up -d
```

## 🔍 故障排查

### 问题 1: 容器启动后立即退出

**原因**: 共享内存不足

**解决**:
```bash
# 添加 --shm-size 参数
docker run --shm-size=2g ...
```

### 问题 2: 无法连接到 9222 端口

**检查清单**:

1. 容器是否在运行？
```bash
docker ps | grep chrome
```

2. 端口是否正确映射？
```bash
docker port browserpilot-chrome
# 应该显示: 9222/tcp -> 0.0.0.0:9222
```

3. 防火墙是否阻止？
```bash
sudo ufw allow 9222/tcp
```

4. 测试连接：
```bash
curl http://localhost:9222/json/version
```

### 问题 3: Chrome 占用内存过高

**解决**: 限制内存使用

```bash
docker run --memory=2g --memory-swap=2g ...
```

或在 docker-compose.yml 中：

```yaml
deploy:
  resources:
    limits:
      memory: 2G
```

### 问题 4: BrowserPilot 连接失败

**日志输出**:
```
[ERROR] Failed to connect browser: timeout
```

**解决步骤**:

1. 确认 Chrome 容器正在运行
2. 测试 URL 是否可访问：`curl http://localhost:9222/json/version`
3. 检查 config.toml 中的 control_url 是否正确
4. 如果在 Docker 网络中，使用容器名而不是 localhost

## 📝 常用命令

```bash
# 查看日志
docker logs -f browserpilot-chrome

# 查看资源使用
docker stats browserpilot-chrome

# 停止容器
docker stop browserpilot-chrome

# 启动容器
docker start browserpilot-chrome

# 重启容器
docker restart browserpilot-chrome

# 删除容器
docker rm -f browserpilot-chrome

# 进入容器
docker exec -it browserpilot-chrome sh

# 查看容器详情
docker inspect browserpilot-chrome
```

## 🌐 访问远程 Chrome

如果 Chrome 运行在远程服务器上：

1. 启动 Chrome 并允许外部访问：
```bash
docker run -d \
  --name browserpilot-chrome \
  -p 9222:9222 \
  --shm-size=2g \
  zenika/alpine-chrome:latest \
  --no-sandbox \
  --remote-debugging-address=0.0.0.0 \
  --remote-debugging-port=9222
```

2. 配置 BrowserPilot：
```toml
[browser]
control_url = "http://192.168.1.100:9222"  # 替换为实际 IP
```

3. 确保防火墙允许访问：
```bash
sudo ufw allow 9222/tcp
```

## 🔒 安全建议

1. **不要暴露到公网**: 仅在可信网络中使用
2. **使用防火墙**: 限制访问来源
3. **容器隔离**: 使用 Docker 网络隔离
4. **定期更新**: 保持镜像最新

```bash
# 更新镜像
docker pull zenika/alpine-chrome:latest
docker stop browserpilot-chrome
docker rm browserpilot-chrome
# 重新运行启动命令
```

## 💡 进阶技巧

### 1. 持久化用户数据

```bash
docker run -d \
  --name browserpilot-chrome \
  -p 9222:9222 \
  --shm-size=2g \
  -v chrome-data:/home/chrome/.config/chromium \
  zenika/alpine-chrome:latest \
  ...
```

### 2. 使用代理

```bash
docker run -d \
  --name browserpilot-chrome \
  -p 9222:9222 \
  --shm-size=2g \
  -e http_proxy=http://proxy.com:8080 \
  -e https_proxy=http://proxy.com:8080 \
  zenika/alpine-chrome:latest \
  ...
```

### 3. VNC 可视化（调试用）

```bash
docker run -d \
  --name browserpilot-chrome-vnc \
  -p 9222:9222 \
  -p 5900:5900 \
  --shm-size=2g \
  siomiz/chrome
```

然后使用 VNC 客户端连接: `vnc://localhost:5900`

## 📚 更多资源

- [完整文档](README.md)
- [远程 Chrome 配置指南](../../REMOTE_CHROME_SETUP.md)
- [测试脚本](test-chrome.sh)
- [启动脚本](start-chrome.sh)

## ❓ 需要帮助？

如果遇到问题：

1. 运行测试脚本: `./test-chrome.sh`
2. 查看容器日志: `docker logs browserpilot-chrome`
3. 检查[完整文档](README.md)中的故障排查部分

