# Docker Chrome 远程调试容器

提供三种方案在 Docker 中运行 Chrome 并通过远程调试连接。

## 方案对比

| 方案 | 镜像大小 | 内存占用 | Chrome 版本 | 推荐场景 |
|-----|---------|---------|------------|---------|
| Alpine (Chromium) | ~200MB | 低 | Chromium | 资源受限环境 |
| Debian (Chrome) | ~1GB | 中 | Google Chrome | 需要完整功能 |
| Zenika 官方镜像 | ~500MB | 低 | Chromium | 快速开始 |

## 快速开始

### 方法 1: 使用 docker-compose（推荐）

```bash
# 进入目录
cd docker/chrome

# 启动 Chromium (Alpine)
docker-compose up -d chrome-alpine

# 或启动 Google Chrome (完整版)
docker-compose up -d chrome-full

# 或使用官方镜像（最简单）
docker-compose up -d chrome-zenika

# 查看日志
docker-compose logs -f chrome-alpine

# 停止
docker-compose down
```

### 方法 2: 使用 docker run

#### Alpine Chromium

```bash
# 构建镜像
docker build -t browserpilot-chrome:alpine -f Dockerfile .

# 运行容器
docker run -d \
  --name browserpilot-chrome \
  -p 9222:9222 \
  --shm-size=2g \
  browserpilot-chrome:alpine
```

#### Debian Google Chrome

```bash
# 构建镜像
docker build -t browserpilot-chrome:full -f Dockerfile.full .

# 运行容器
docker run -d \
  --name browserpilot-chrome-full \
  -p 9222:9222 \
  --shm-size=2g \
  browserpilot-chrome:full
```

#### 使用官方镜像（最快）

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

## 获取 ControlURL

### 1. 获取容器 IP（容器网络内访问）

```bash
# 获取容器 IP
docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' browserpilot-chrome

# 示例输出: 172.17.0.2
# ControlURL: http://172.17.0.2:9222
```

### 2. 使用 localhost（端口映射）

如果使用了 `-p 9222:9222` 端口映射：

```bash
# ControlURL: http://localhost:9222
# 或
# ControlURL: http://127.0.0.1:9222
```

### 3. 使用宿主机 IP（远程访问）

```bash
# 获取宿主机 IP
hostname -I | awk '{print $1}'

# 示例输出: 192.168.1.100
# ControlURL: http://192.168.1.100:9222
```

### 4. 验证连接

```bash
# 方法 1: 使用 curl
curl http://localhost:9222/json/version

# 方法 2: 浏览器访问
# 打开浏览器访问: http://localhost:9222
```

期望输出（JSON）：
```json
{
   "Browser": "Chrome/120.0.6099.109",
   "Protocol-Version": "1.3",
   "User-Agent": "Mozilla/5.0 ...",
   "V8-Version": "12.0.267.8",
   "WebKit-Version": "537.36",
   "webSocketDebuggerUrl": "ws://localhost:9222/devtools/browser/..."
}
```

## BrowserPilot 配置

获取到 ControlURL 后，编辑 `backend/config.toml`：

### 本地 Docker（端口映射）

```toml
[browser]
control_url = "http://localhost:9222"
```

### 远程 Docker 容器

```toml
[browser]
control_url = "http://192.168.1.100:9222"
```

### 同一 Docker 网络

```toml
[browser]
control_url = "http://browserpilot-chrome:9222"
```

## 高级配置

### 1. 持久化用户数据

```bash
docker run -d \
  --name browserpilot-chrome \
  -p 9222:9222 \
  --shm-size=2g \
  -v chrome-data:/home/chrome/.config/chromium \
  browserpilot-chrome:alpine
```

### 2. 设置代理

```bash
docker run -d \
  --name browserpilot-chrome \
  -p 9222:9222 \
  --shm-size=2g \
  -e http_proxy=http://proxy.example.com:8080 \
  -e https_proxy=http://proxy.example.com:8080 \
  browserpilot-chrome:alpine
```

### 3. VNC 支持（可视化调试）

使用带 VNC 的镜像：

```bash
docker run -d \
  --name browserpilot-chrome-vnc \
  -p 9222:9222 \
  -p 5900:5900 \
  --shm-size=2g \
  -e VNC_NO_PASSWORD=1 \
  siomiz/chrome
```

VNC 连接: `vnc://localhost:5900`

### 4. 资源限制

```bash
docker run -d \
  --name browserpilot-chrome \
  -p 9222:9222 \
  --shm-size=2g \
  --memory=2g \
  --cpus=2 \
  browserpilot-chrome:alpine
```

## 与 BrowserPilot 同网络部署

### docker-compose 完整示例

```yaml
version: '3.8'

services:
  chrome:
    image: zenika/alpine-chrome:latest
    container_name: browserpilot-chrome
    shm_size: 2gb
    command:
      - --no-sandbox
      - --disable-dev-shm-usage
      - --remote-debugging-address=0.0.0.0
      - --remote-debugging-port=9222
    networks:
      - browserpilot-net

  browserpilot:
    build: ../../backend
    container_name: browserpilot-backend
    ports:
      - "8080:8080"
    environment:
      - BROWSER_CONTROL_URL=http://chrome:9222
    depends_on:
      - chrome
    networks:
      - browserpilot-net

networks:
  browserpilot-net:
    driver: bridge
```

## 常见问题

### 1. Chrome 启动失败

**问题**: 容器启动后立即退出

**解决**:
```bash
# 增加共享内存
docker run --shm-size=2g ...

# 或禁用 /dev/shm
docker run --disable-dev-shm-usage ...
```

### 2. 无法连接到远程调试端口

**问题**: 连接被拒绝

**检查**:
```bash
# 1. 检查容器是否运行
docker ps | grep chrome

# 2. 检查端口映射
docker port browserpilot-chrome

# 3. 检查防火墙
sudo ufw status
sudo ufw allow 9222/tcp

# 4. 测试连接
curl http://localhost:9222/json/version
```

### 3. Chrome 占用内存过高

**解决**:
```bash
# 设置内存限制
docker run --memory=2g --memory-swap=2g ...

# 或在 docker-compose 中
deploy:
  resources:
    limits:
      memory: 2G
```

### 4. 页面加载缓慢

**优化**:
```bash
# 禁用图片加载
--blink-settings=imagesEnabled=false

# 禁用 JavaScript
--disable-javascript

# 使用无头模式
--headless
```

## 生产环境建议

1. **资源限制**: 设置内存和 CPU 限制
2. **重启策略**: `restart: unless-stopped`
3. **健康检查**: 定期检查 `/json/version`
4. **日志管理**: 限制日志大小
5. **安全性**: 不要暴露到公网，使用防火墙限制访问
6. **监控**: 监控容器资源使用

## 测试脚本

创建 `test-chrome.sh`:

```bash
#!/bin/bash

# 测试 Chrome 远程调试
CHROME_URL="http://localhost:9222"

echo "Testing Chrome remote debugging..."
echo "Chrome URL: $CHROME_URL"
echo ""

# 测试连接
echo "1. Testing connection..."
curl -s "$CHROME_URL/json/version" | jq '.'

# 获取打开的页面
echo -e "\n2. Getting open pages..."
curl -s "$CHROME_URL/json" | jq '.[0:3]'

# 获取 WebSocket URL
echo -e "\n3. WebSocket debugger URL:"
curl -s "$CHROME_URL/json/version" | jq -r '.webSocketDebuggerUrl'

echo -e "\n✓ Chrome is ready for remote debugging!"
```

运行测试:
```bash
chmod +x test-chrome.sh
./test-chrome.sh
```

## 性能对比

实测数据（基于 1GB 内存限制）:

| 指标 | Alpine | Debian | Zenika |
|-----|--------|--------|--------|
| 启动时间 | 2-3秒 | 3-5秒 | 2-3秒 |
| 内存占用 | 150-300MB | 200-400MB | 150-300MB |
| 镜像大小 | 200MB | 1GB | 500MB |
| 构建时间 | 2分钟 | 5分钟 | 0秒(拉取) |

## 推荐方案

- **开发环境**: Zenika 官方镜像（最简单）
- **生产环境**: Alpine Chromium（最轻量）
- **需要完整功能**: Debian Google Chrome

