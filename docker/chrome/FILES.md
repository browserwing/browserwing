# Docker Chrome 文件说明

本目录包含在 Docker 中运行 Chrome 并连接到 BrowserPilot 的完整解决方案。

## 📁 文件列表

### 核心文件

| 文件 | 说明 | 用途 |
|-----|------|------|
| `Dockerfile` | Alpine Chromium 镜像 | 轻量级 Chrome 容器（200MB） |
| `Dockerfile.full` | Debian Chrome 镜像 | 完整 Chrome 容器（1GB） |
| `docker-compose.yml` | Docker Compose 配置 | 一键启动多种 Chrome 方案 |

### 脚本文件

| 文件 | 说明 | 使用方法 |
|-----|------|---------|
| `start-chrome.sh` | 🚀 一键启动脚本 | `./start-chrome.sh` |
| `test-chrome.sh` | ✅ 连接测试脚本 | `./test-chrome.sh` |

### 文档文件

| 文件 | 说明 | 适合人群 |
|-----|------|---------|
| `QUICKSTART.md` | ⚡ 5分钟快速开始 | 新手、快速上手 |
| `README.md` | 📚 完整文档 | 深入了解、生产部署 |
| `FILES.md` | 📋 本文件 | 了解文件结构 |

## 🚀 快速开始

### 方法 1: 使用启动脚本（推荐）

```bash
# 进入目录
cd docker/chrome

# 运行启动脚本
./start-chrome.sh

# 选择选项 1（使用官方镜像）
```

### 方法 2: 直接使用 Docker

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

### 方法 3: 使用 docker-compose

```bash
# 启动 Zenika 官方镜像
docker-compose up -d chrome-zenika

# 或启动 Alpine 镜像
docker-compose up -d chrome-alpine

# 或启动完整版 Chrome
docker-compose up -d chrome-full
```

## 📖 文档导航

### 我是新手，想快速上手
👉 阅读 [QUICKSTART.md](QUICKSTART.md)

### 我想了解所有细节
👉 阅读 [README.md](README.md)

### 我想了解远程 Chrome 配置
👉 阅读 [../../REMOTE_CHROME_SETUP.md](../../REMOTE_CHROME_SETUP.md)

### 我想测试连接
👉 运行 `./test-chrome.sh`

### 我想自定义配置
👉 编辑 `docker-compose.yml` 或 `Dockerfile`

## 🎯 使用场景

### 场景 1: 开发环境（最简单）

```bash
./start-chrome.sh
# 选择选项 1
```

配置 `backend/config.toml`:
```toml
[browser]
control_url = "http://localhost:9222"
```

### 场景 2: 生产环境（推荐）

使用 `docker-compose.yml`:

```yaml
version: '3.8'
services:
  chrome:
    image: zenika/alpine-chrome:latest
    shm_size: 2gb
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 2G
```

### 场景 3: 自定义镜像

编辑 `Dockerfile` 或 `Dockerfile.full`，然后：

```bash
docker build -t my-chrome:latest -f Dockerfile .
docker run -d --name my-chrome -p 9222:9222 --shm-size=2g my-chrome:latest
```

## 🔧 常见任务

### 查看日志
```bash
docker logs -f browserpilot-chrome
```

### 测试连接
```bash
./test-chrome.sh
# 或
curl http://localhost:9222/json/version
```

### 获取 Control URL
```bash
# 本地
echo "http://localhost:9222"

# 容器 IP
docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' browserpilot-chrome

# 主机 IP
hostname -I | awk '{print $1}'
```

### 停止/启动/重启
```bash
docker stop browserpilot-chrome
docker start browserpilot-chrome
docker restart browserpilot-chrome
```

### 删除容器
```bash
docker rm -f browserpilot-chrome
```

### 更新镜像
```bash
docker pull zenika/alpine-chrome:latest
docker stop browserpilot-chrome
docker rm browserpilot-chrome
# 重新运行启动命令
```

## 📊 方案对比

| 特性 | Zenika 官方 | Alpine 自建 | Debian 完整 |
|-----|------------|------------|------------|
| 镜像大小 | 500MB | 200MB | 1GB |
| 启动速度 | ⚡⚡⚡ | ⚡⚡⚡ | ⚡⚡ |
| 内存占用 | 低 | 低 | 中 |
| 功能完整度 | 高 | 中 | 最高 |
| 维护成本 | 无需维护 | 需要维护 | 需要维护 |
| 推荐场景 | 开发/生产 | 资源受限 | 需要完整功能 |
| 文件 | 无需构建 | `Dockerfile` | `Dockerfile.full` |

## 💡 最佳实践

### 1. 开发环境
- 使用 Zenika 官方镜像
- 使用 `start-chrome.sh` 快速启动
- 端口映射到 localhost

### 2. 生产环境
- 使用 docker-compose
- 设置资源限制
- 配置重启策略
- 使用 Docker 网络隔离
- 定期更新镜像

### 3. 性能优化
- 设置合适的 `--shm-size`
- 限制内存使用
- 禁用不需要的功能
- 使用持久化卷

### 4. 安全建议
- 不要暴露到公网
- 使用防火墙限制访问
- 定期更新镜像
- 使用非 root 用户运行

## 🐛 故障排查

### 容器无法启动
```bash
# 检查日志
docker logs browserpilot-chrome

# 增加共享内存
docker run --shm-size=2g ...
```

### 无法连接
```bash
# 检查容器状态
docker ps | grep chrome

# 检查端口
docker port browserpilot-chrome

# 测试连接
curl http://localhost:9222/json/version
```

### 内存占用高
```bash
# 限制内存
docker run --memory=2g ...

# 查看资源使用
docker stats browserpilot-chrome
```

## 🔗 相关链接

- [BrowserPilot 主项目](../../README.md)
- [远程 Chrome 配置指南](../../REMOTE_CHROME_SETUP.md)
- [Zenika Alpine Chrome](https://github.com/Zenika/alpine-chrome)
- [Rod 浏览器自动化](https://go-rod.github.io/)

## 📝 版本历史

- v1.0 - 初始版本
  - 提供三种 Dockerfile
  - docker-compose 支持
  - 启动和测试脚本
  - 完整文档

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

与 BrowserPilot 主项目相同。

