.PHONY: help install dev build clean backend frontend build-embedded build-linux build-windows build-all package

# 应用信息
APP_NAME = browserwing
VERSION = "v0.0.1"
BUILD_TIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION = $(shell go version | awk '{print $$3}')

# 默认端口配置
BACKEND_PORT ?= 8080
FRONTEND_PORT ?= 5173

# 目录配置
BUILD_DIR = build
FRONTEND_DIR = frontend
BACKEND_DIR = backend
DIST_DIR = $(BACKEND_DIR)/dist

# Go 构建参数
LDFLAGS = -ldflags "-s -w \
	-X 'main.Version=$(VERSION)' \
	-X 'main.BuildTime=$(BUILD_TIME)' \
	-X 'main.GoVersion=$(GO_VERSION)'"

# 构建标签
BUILD_TAGS = -tags embed

# 颜色输出
COLOR_RESET = \033[0m
COLOR_GREEN = \033[0;32m
COLOR_YELLOW = \033[1;33m
COLOR_BLUE = \033[0;34m

help:
	@echo "$(COLOR_BLUE)BrowserWing - 智能自动写作工具$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_GREEN)开发命令:$(COLOR_RESET)"
	@echo "  make install              - 安装所有依赖"
	@echo "  make dev                  - 启动开发环境（分离前后端）"
	@echo "  make backend              - 仅运行后端"
	@echo "  make frontend             - 仅运行前端"
	@echo ""
	@echo "$(COLOR_GREEN)构建命令:$(COLOR_RESET)"
	@echo "  make build-embedded       - 构建当前平台的集成版本（前端嵌入后端）"
	@echo "  make build-linux          - 构建 Linux 版本（amd64 和 arm64）"
	@echo "  make build-windows        - 构建 Windows 版本（amd64 和 arm64）"
	@echo "  make build-all            - 构建所有平台的集成版本"
	@echo "  make package              - 打包所有平台并生成压缩包"
	@echo ""
	@echo "$(COLOR_GREEN)其他命令:$(COLOR_RESET)"
	@echo "  make clean                - 清理构建文件"
	@echo "  make test                 - 运行测试"
	@echo "  make fmt                  - 格式化代码"
	@echo ""
	@echo "$(COLOR_YELLOW)自定义端口:$(COLOR_RESET)"
	@echo "  make dev BACKEND_PORT=3000 FRONTEND_PORT=5000"
	@echo "  make backend BACKEND_PORT=3000"

install:
	@echo "$(COLOR_YELLOW)📦 安装后端依赖...$(COLOR_RESET)"
	cd backend && go mod download
	@echo "$(COLOR_YELLOW)📦 安装前端依赖...$(COLOR_RESET)"
	cd frontend && pnpm install
	@echo "$(COLOR_GREEN)✅ 依赖安装完成！$(COLOR_RESET)"

dev:
	@echo "$(COLOR_YELLOW)🚀 启动开发环境...$(COLOR_RESET)"
	@bash start.sh $(BACKEND_PORT) $(FRONTEND_PORT)

backend:
	@echo "$(COLOR_YELLOW)📦 启动后端服务（端口: $(BACKEND_PORT)）...$(COLOR_RESET)"
	cd backend && go run main.go --port $(BACKEND_PORT)

frontend:
	@echo "$(COLOR_YELLOW)🎨 启动前端服务（端口: $(FRONTEND_PORT)）...$(COLOR_RESET)"
	cd frontend && VITE_API_PORT=$(BACKEND_PORT) pnpm dev --port $(FRONTEND_PORT)

# 创建构建目录
$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

# 构建前端
build-frontend:
	@echo "$(COLOR_YELLOW)构建前端...$(COLOR_RESET)"
	@if [ ! -d "$(FRONTEND_DIR)/node_modules" ]; then \
		echo "$(COLOR_BLUE)安装前端依赖...$(COLOR_RESET)"; \
		cd $(FRONTEND_DIR) && pnpm install; \
	fi
	@cd $(FRONTEND_DIR) && pnpm build
	@echo "$(COLOR_GREEN)✓ 前端构建完成$(COLOR_RESET)"

# 复制前端产物到后端目录
copy-frontend: build-frontend
	@echo "$(COLOR_YELLOW)复制前端产物到后端...$(COLOR_RESET)"
	@rm -rf $(DIST_DIR)
	@cp -r $(FRONTEND_DIR)/dist $(DIST_DIR)
	@echo "$(COLOR_GREEN)✓ 复制完成$(COLOR_RESET)"

# 旧版构建命令（兼容）
build:
	@echo "$(COLOR_YELLOW)🔨 构建后端...$(COLOR_RESET)"
	cd backend && go build -o ../bin/browserwing .
	@echo "$(COLOR_YELLOW)🔨 构建前端...$(COLOR_RESET)"
	cd frontend && pnpm build
	@echo "$(COLOR_GREEN)✅ 构建完成！$(COLOR_RESET)"
	@echo "   后端: bin/browserwing"
	@echo "   前端: frontend/dist"

# 构建当前平台的集成版本
build-embedded: $(BUILD_DIR) copy-frontend
	@echo "$(COLOR_YELLOW)🔨 构建当前平台集成版本...$(COLOR_RESET)"
	@cd $(BACKEND_DIR) && go build $(BUILD_TAGS) $(LDFLAGS) -o ../$(BUILD_DIR)/$(APP_NAME) .
	@echo "$(COLOR_GREEN)✓ 构建完成: $(BUILD_DIR)/$(APP_NAME)$(COLOR_RESET)"
	@echo "$(COLOR_BLUE)💡 运行: ./$(BUILD_DIR)/$(APP_NAME) --port 8080$(COLOR_RESET)"

build-mac: $(BUILD_DIR) copy-frontend
	@echo "$(COLOR_YELLOW)🍎 构建 macOS 版本...$(COLOR_RESET)"
	@cd $(BACKEND_DIR) && GOOS=darwin GOARCH=amd64 go build $(BUILD_TAGS) $(LDFLAGS) \
		-o ../$(BUILD_DIR)/$(APP_NAME)-mac-amd64 .
	@echo "$(COLOR_GREEN)✓ macOS amd64: $(BUILD_DIR)/$(APP_NAME)-mac-amd64$(COLOR_RESET)"
	@cd $(BACKEND_DIR) && GOOS=darwin GOARCH=arm64 go build $(BUILD_TAGS) $(LDFLAGS) \
		-o ../$(BUILD_DIR)/$(APP_NAME)-mac-arm64 .
	@echo "$(COLOR_GREEN)✓ macOS arm64: $(BUILD_DIR)/$(APP_NAME)-mac-arm64$(COLOR_RESET)"

# 构建 Linux 版本
build-linux: $(BUILD_DIR) copy-frontend
	@echo "$(COLOR_YELLOW)🐧 构建 Linux 版本...$(COLOR_RESET)"
	@cd $(BACKEND_DIR) && GOOS=linux GOARCH=amd64 go build $(BUILD_TAGS) $(LDFLAGS) \
		-o ../$(BUILD_DIR)/$(APP_NAME)-linux-amd64 .
	@echo "$(COLOR_GREEN)✓ Linux amd64: $(BUILD_DIR)/$(APP_NAME)-linux-amd64$(COLOR_RESET)"
	@cd $(BACKEND_DIR) && GOOS=linux GOARCH=arm64 go build $(BUILD_TAGS) $(LDFLAGS) \
		-o ../$(BUILD_DIR)/$(APP_NAME)-linux-arm64 .
	@echo "$(COLOR_GREEN)✓ Linux arm64: $(BUILD_DIR)/$(APP_NAME)-linux-arm64$(COLOR_RESET)"

# 构建 Windows 版本
build-windows: $(BUILD_DIR) copy-frontend 
	@$(MAKE) build-windows-amd64
	@$(MAKE) build-windows-arm64	

build-windows-arm64: copy-frontend
	@echo "$(COLOR_YELLOW)🪟 构建 Windows arm64 版本...$(COLOR_RESET)"
	@cd $(BACKEND_DIR) && GOOS=windows GOARCH=arm64 go build $(BUILD_TAGS) $(LDFLAGS) \
		-o ../$(BUILD_DIR)/$(APP_NAME)-windows-arm64.exe .
	@echo "$(COLOR_GREEN)✓ Windows arm64: $(BUILD_DIR)/$(APP_NAME)-windows-arm64.exe$(COLOR_RESET)"

build-windows-amd64: copy-frontend
	@echo "$(COLOR_YELLOW)🪟 构建 Windows amd64 版本...$(COLOR_RESET)"
	@cd $(BACKEND_DIR) && GOOS=windows GOARCH=amd64 go build $(BUILD_TAGS) $(LDFLAGS) \
		-o ../$(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe .
	@echo "$(COLOR_GREEN)✓ Windows amd64: $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe$(COLOR_RESET)"	

# 构建所有平台
build-all: build-linux build-windows build-mac
	@echo ""
	@echo "$(COLOR_GREEN)✅ 所有平台构建完成!$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BLUE)📦 构建产物:$(COLOR_RESET)"
	@ls -lh $(BUILD_DIR)/ | grep -v "^total" | awk '{printf "  %-40s %10s\n", $$9, $$5}'
	@echo ""
	@echo "$(COLOR_YELLOW)使用说明:$(COLOR_RESET)"
	@echo "  Linux:   ./$(BUILD_DIR)/$(APP_NAME)-linux-amd64 --port 8080"
	@echo "  Windows: $(BUILD_DIR)\\$(APP_NAME)-windows-amd64.exe --port 8080"

# 打包发布（生成压缩包）
package: build-all
	@echo "$(COLOR_YELLOW)📦 打包发布版本...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)/releases
	@cd $(BUILD_DIR) && tar -czf releases/$(APP_NAME)-linux-amd64-$(VERSION).tar.gz $(APP_NAME)-linux-amd64
	@cd $(BUILD_DIR) && tar -czf releases/$(APP_NAME)-linux-arm64-$(VERSION).tar.gz $(APP_NAME)-linux-arm64
	@cd $(BUILD_DIR) && zip -q releases/$(APP_NAME)-windows-amd64-$(VERSION).zip $(APP_NAME)-windows-amd64.exe
	@cd $(BUILD_DIR) && zip -q releases/$(APP_NAME)-windows-arm64-$(VERSION).zip $(APP_NAME)-windows-arm64.exe
	@echo "$(COLOR_GREEN)✓ 发布包已创建:$(COLOR_RESET)"
	@ls -lh $(BUILD_DIR)/releases/ | grep -v "^total" | awk '{printf "  %-50s %10s\n", $$9, $$5}'

# 运行集成版本
run: build-embedded
	@echo "$(COLOR_YELLOW)🚀 启动集成版本...$(COLOR_RESET)"
	@$(BUILD_DIR)/$(APP_NAME) --port $(BACKEND_PORT)

# 测试
test:
	@echo "$(COLOR_YELLOW)🧪 运行测试...$(COLOR_RESET)"
	@cd $(BACKEND_DIR) && go test -v ./...
	@echo "$(COLOR_GREEN)✓ 测试完成$(COLOR_RESET)"

# 格式化代码
fmt:
	@echo "$(COLOR_YELLOW)📝 格式化代码...$(COLOR_RESET)"
	@cd $(BACKEND_DIR) && go fmt ./...
	@echo "$(COLOR_GREEN)✓ 代码格式化完成$(COLOR_RESET)"

# 显示版本信息
version:
	@echo "应用名称:    $(APP_NAME)"
	@echo "版本:        $(VERSION)"
	@echo "构建时间:    $(BUILD_TIME)"
	@echo "Go 版本:     $(GO_VERSION)"


clean:
	@echo "$(COLOR_YELLOW)🧹 清理构建文件...$(COLOR_RESET)"
	@rm -rf bin/
	@rm -rf $(BUILD_DIR)/
	@rm -rf $(DIST_DIR)
	@rm -rf frontend/dist
	@rm -rf backend/data
	@echo "$(COLOR_GREEN)✅ 清理完成！$(COLOR_RESET)"

