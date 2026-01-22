#!/bin/bash

# Chrome 远程调试测试脚本
# 用途：验证 Chrome 容器是否正常运行并可以接受远程调试连接

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 默认配置
CHROME_URL="${CHROME_URL:-http://localhost:9222}"
TIMEOUT=30

echo "════════════════════════════════════════════════════════"
echo "  Chrome 远程调试测试工具"
echo "════════════════════════════════════════════════════════"
echo ""
echo "Chrome URL: $CHROME_URL"
echo "Timeout: ${TIMEOUT}s"
echo ""

# 函数：打印成功消息
success() {
    echo -e "${GREEN}✓${NC} $1"
}

# 函数：打印错误消息
error() {
    echo -e "${RED}✗${NC} $1"
}

# 函数：打印警告消息
warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# 函数：打印信息
info() {
    echo -e "ℹ $1"
}

# 测试 1: 基础连接测试
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "测试 1: 基础连接测试"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if curl -s --connect-timeout $TIMEOUT "$CHROME_URL/json/version" > /dev/null 2>&1; then
    success "Chrome 远程调试端口可访问"
else
    error "无法连接到 Chrome 远程调试端口"
    error "请检查："
    echo "  1. Chrome 容器是否正在运行: docker ps | grep chrome"
    echo "  2. 端口映射是否正确: docker port <container-name>"
    echo "  3. 防火墙设置是否允许访问"
    exit 1
fi
echo ""

# 测试 2: 获取版本信息
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "测试 2: Chrome 版本信息"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

VERSION_INFO=$(curl -s "$CHROME_URL/json/version")

if command -v jq &> /dev/null; then
    BROWSER=$(echo "$VERSION_INFO" | jq -r '.Browser')
    PROTOCOL=$(echo "$VERSION_INFO" | jq -r '."Protocol-Version"')
    WEBKIT=$(echo "$VERSION_INFO" | jq -r '."WebKit-Version"')
    WS_URL=$(echo "$VERSION_INFO" | jq -r '.webSocketDebuggerUrl')
    
    success "浏览器版本: $BROWSER"
    info "协议版本: $PROTOCOL"
    info "WebKit 版本: $WEBKIT"
    echo ""
    info "WebSocket URL: $WS_URL"
else
    warning "未安装 jq，显示原始 JSON："
    echo "$VERSION_INFO"
fi
echo ""

# 测试 3: 获取打开的页面
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "测试 3: 打开的页面列表"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

PAGES_INFO=$(curl -s "$CHROME_URL/json")
PAGE_COUNT=$(echo "$PAGES_INFO" | jq '. | length' 2>/dev/null || echo "unknown")

if [ "$PAGE_COUNT" != "unknown" ]; then
    success "检测到 $PAGE_COUNT 个页面"
    
    if [ "$PAGE_COUNT" -gt 0 ]; then
        echo ""
        info "前 3 个页面信息："
        echo "$PAGES_INFO" | jq -r '.[:3] | .[] | "  - [\(.type)] \(.title) - \(.url)"' 2>/dev/null || echo "$PAGES_INFO"
    fi
else
    warning "无法解析页面数量"
    echo "$PAGES_INFO"
fi
echo ""

# 测试 4: 测试创建新页面
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "测试 4: 创建新页面测试"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

NEW_PAGE=$(curl -s "$CHROME_URL/json/new?about:blank")
if [ -n "$NEW_PAGE" ]; then
    success "成功创建新页面"
    
    if command -v jq &> /dev/null; then
        PAGE_ID=$(echo "$NEW_PAGE" | jq -r '.id')
        info "页面 ID: $PAGE_ID"
        
        # 关闭测试页面
        sleep 1
        CLOSE_RESULT=$(curl -s "$CHROME_URL/json/close/$PAGE_ID")
        if [ "$CLOSE_RESULT" == "Target is closing" ]; then
            success "成功关闭测试页面"
        fi
    fi
else
    error "创建新页面失败"
fi
echo ""

# 测试 5: WebSocket 连接测试
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "测试 5: WebSocket 连接测试"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if command -v wscat &> /dev/null; then
    if [ -n "$WS_URL" ]; then
        success "WebSocket URL 可用"
        info "可以使用以下命令测试 WebSocket 连接："
        echo "  wscat -c '$WS_URL'"
    else
        warning "无法获取 WebSocket URL"
    fi
else
    warning "未安装 wscat，跳过 WebSocket 测试"
    info "安装方法: npm install -g wscat"
fi
echo ""

# 测试总结
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "测试总结"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
success "所有基础测试通过！"
echo ""
info "BrowserPilot 配置示例："
echo ""
echo "[browser]"
echo "control_url = \"$CHROME_URL\""
echo ""
success "Chrome 已准备好接受远程调试连接！"
echo ""

# Docker 相关信息
if command -v docker &> /dev/null; then
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Docker 容器信息"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    CONTAINER_ID=$(docker ps --filter "publish=9222" --format "{{.ID}}" | head -1)
    
    if [ -n "$CONTAINER_ID" ]; then
        CONTAINER_NAME=$(docker inspect --format='{{.Name}}' "$CONTAINER_ID" | sed 's/\///')
        CONTAINER_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$CONTAINER_ID")
        CONTAINER_STATUS=$(docker inspect --format='{{.State.Status}}' "$CONTAINER_ID")
        CONTAINER_MEMORY=$(docker stats --no-stream --format "{{.MemUsage}}" "$CONTAINER_ID")
        
        info "容器名称: $CONTAINER_NAME"
        info "容器 ID: $CONTAINER_ID"
        info "容器 IP: $CONTAINER_IP"
        info "运行状态: $CONTAINER_STATUS"
        info "内存使用: $CONTAINER_MEMORY"
        echo ""
        
        info "其他可用的 Control URL："
        echo "  - http://localhost:9222"
        echo "  - http://$CONTAINER_IP:9222"
        echo "  - http://$(hostname -I | awk '{print $1}'):9222"
    else
        warning "未找到监听 9222 端口的 Docker 容器"
    fi
    echo ""
fi

exit 0

