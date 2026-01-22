#!/bin/bash

# Chrome Docker 容器快速启动脚本
# 用途：一键启动 Chrome 并获取 ControlURL

set -e

# 颜色输出
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "════════════════════════════════════════════════════════"
echo "  BrowserPilot - Chrome Docker 快速启动"
echo "════════════════════════════════════════════════════════"
echo ""

# 检查 Docker 是否已安装
if ! command -v docker &> /dev/null; then
    echo -e "${YELLOW}错误: 未安装 Docker${NC}"
    echo "请先安装 Docker: https://docs.docker.com/get-docker/"
    exit 1
fi

# 选择启动方式
echo "请选择启动方式："
echo "  1) 使用官方镜像 (Zenika/alpine-chrome) - 推荐，最快"
echo "  2) 使用 docker-compose - 完整方案"
echo "  3) 构建自定义 Alpine 镜像 - 轻量级"
echo "  4) 构建完整 Debian 镜像 - 功能最全"
echo ""
read -p "请输入选项 (1-4) [默认: 1]: " choice
choice=${choice:-1}

CONTAINER_NAME="browserpilot-chrome"
PORT=9222

case $choice in
    1)
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "使用官方镜像启动 Chrome..."
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        
        # 停止并删除已存在的容器
        if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
            echo "停止已存在的容器..."
            docker stop $CONTAINER_NAME >/dev/null 2>&1 || true
            docker rm $CONTAINER_NAME >/dev/null 2>&1 || true
        fi
        
        echo "拉取镜像..."
        docker pull zenika/alpine-chrome:latest
        
        echo "启动容器..."
        docker run -d \
          --name $CONTAINER_NAME \
          -p $PORT:9222 \
          --shm-size=2g \
          --restart unless-stopped \
          zenika/alpine-chrome:latest \
          --no-sandbox \
          --disable-dev-shm-usage \
          --remote-debugging-address=0.0.0.0 \
          --remote-debugging-port=9222 \
          --no-first-run \
          --disable-extensions
        ;;
        
    2)
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "使用 docker-compose 启动..."
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        
        if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null 2>&1; then
            echo -e "${YELLOW}错误: 未安装 docker-compose${NC}"
            exit 1
        fi
        
        # 使用 docker compose 或 docker-compose
        if docker compose version &> /dev/null 2>&1; then
            COMPOSE_CMD="docker compose"
        else
            COMPOSE_CMD="docker-compose"
        fi
        
        $COMPOSE_CMD up -d chrome-zenika
        CONTAINER_NAME="browserpilot-chrome-zenika"
        PORT=9224
        ;;
        
    3)
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "构建 Alpine 镜像..."
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        
        docker build -t browserpilot-chrome:alpine -f Dockerfile .
        
        if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
            docker stop $CONTAINER_NAME >/dev/null 2>&1 || true
            docker rm $CONTAINER_NAME >/dev/null 2>&1 || true
        fi
        
        docker run -d \
          --name $CONTAINER_NAME \
          -p $PORT:9222 \
          --shm-size=2g \
          --restart unless-stopped \
          browserpilot-chrome:alpine
        ;;
        
    4)
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "构建完整 Debian 镜像（需要较长时间）..."
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        
        docker build -t browserpilot-chrome:full -f Dockerfile.full .
        
        if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
            docker stop $CONTAINER_NAME >/dev/null 2>&1 || true
            docker rm $CONTAINER_NAME >/dev/null 2>&1 || true
        fi
        
        docker run -d \
          --name $CONTAINER_NAME \
          -p $PORT:9222 \
          --shm-size=2g \
          --restart unless-stopped \
          browserpilot-chrome:full
        ;;
        
    *)
        echo "无效的选项"
        exit 1
        ;;
esac

echo ""
echo -e "${GREEN}✓ Chrome 容器启动成功！${NC}"
echo ""

# 等待 Chrome 启动
echo "等待 Chrome 初始化..."
sleep 3

# 获取各种 URL
CONTAINER_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $CONTAINER_NAME)
HOST_IP=$(hostname -I | awk '{print $1}')

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Chrome 容器信息"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${BLUE}容器名称:${NC} $CONTAINER_NAME"
echo -e "${BLUE}容器 ID:${NC} $(docker ps --filter name=$CONTAINER_NAME --format '{{.ID}}')"
echo -e "${BLUE}容器 IP:${NC} $CONTAINER_IP"
echo -e "${BLUE}映射端口:${NC} $PORT"
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "可用的 Control URL"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}本地访问:${NC}"
echo "  http://localhost:$PORT"
echo ""
echo -e "${GREEN}同网络容器访问:${NC}"
echo "  http://$CONTAINER_NAME:9222"
echo "  http://$CONTAINER_IP:9222"
echo ""
echo -e "${GREEN}远程访问:${NC}"
echo "  http://$HOST_IP:$PORT"
echo ""

# 测试连接
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "测试连接"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if curl -s "http://localhost:$PORT/json/version" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Chrome 远程调试端口工作正常${NC}"
    
    if command -v jq &> /dev/null; then
        VERSION=$(curl -s "http://localhost:$PORT/json/version" | jq -r '.Browser')
        echo -e "${BLUE}版本信息:${NC} $VERSION"
    fi
else
    echo -e "${YELLOW}⚠ 无法连接到 Chrome，请稍等片刻后重试${NC}"
fi
echo ""

# BrowserPilot 配置
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "BrowserPilot 配置"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "在 backend/config.toml 中添加："
echo ""
echo -e "${BLUE}[browser]${NC}"
echo -e "${BLUE}control_url = \"http://localhost:$PORT\"${NC}"
echo ""

# 常用命令
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "常用命令"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "查看日志:"
echo "  docker logs -f $CONTAINER_NAME"
echo ""
echo "停止容器:"
echo "  docker stop $CONTAINER_NAME"
echo ""
echo "启动容器:"
echo "  docker start $CONTAINER_NAME"
echo ""
echo "删除容器:"
echo "  docker rm -f $CONTAINER_NAME"
echo ""
echo "进入容器:"
echo "  docker exec -it $CONTAINER_NAME sh"
echo ""
echo "查看资源使用:"
echo "  docker stats $CONTAINER_NAME"
echo ""
echo "运行测试脚本:"
echo "  ./test-chrome.sh"
echo ""

echo -e "${GREEN}════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}Chrome 已准备就绪，可以开始使用！${NC}"
echo -e "${GREEN}════════════════════════════════════════════════════════${NC}"

