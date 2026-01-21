# Headless 模式智能检测功能

## 概述

系统会自动检测运行环境，并设置合适的 headless 模式默认值。这样可以确保：
- 在有 GUI 环境（macOS、Windows、Linux 桌面）中显示浏览器界面，方便调试
- 在无 GUI 环境（Docker、Linux 服务器）中自动启用 headless 模式，避免启动失败

## 检测逻辑

### 检测顺序（从高优先级到低优先级）

#### 1. Docker 容器检测 ⚡ 最高优先级

检查是否运行在 Docker 容器中：
- 检查 `/.dockerenv` 文件是否存在
- 检查 `/proc/1/cgroup` 是否包含 `docker` 或 `containerd` 标识

**结果**：如果检测到 Docker 环境 → **headless = true**

#### 2. 操作系统类型判断 🖥️

根据 `runtime.GOOS` 判断操作系统类型：

| 操作系统 | GOOS 值 | 默认行为 | 原因 |
|---------|---------|---------|------|
| macOS | `darwin` | **headless = false** | macOS 原生支持 GUI，无需 DISPLAY 环境变量 |
| Windows | `windows` | **headless = false** | Windows 原生支持 GUI |
| Linux | `linux` | 继续下一步检测 | 需要检查是否有显示服务器 |

#### 3. Linux 环境 GUI 检测 🐧

仅在 Linux 系统上执行：
- 检查 `DISPLAY` 环境变量（X11 显示服务器）
- 检查 `WAYLAND_DISPLAY` 环境变量（Wayland 显示服务器）

**结果**：
- ✅ 两者都为空 → **headless = true**（无 GUI 环境）
- ❌ 任一存在 → **headless = false**（有 GUI 环境）

### 检测流程图

```
开始
  ↓
检查是否在 Docker 中？
  ↓ 是 → headless = true (结束)
  ↓ 否
检查操作系统类型
  ↓
是 macOS/Windows？
  ↓ 是 → headless = false (结束)
  ↓ 否 (Linux)
检查 DISPLAY 和 WAYLAND_DISPLAY
  ↓
都为空？
  ↓ 是 → headless = true (结束)
  ↓ 否 → headless = false (结束)
```

## 日志输出

系统启动时会输出详细的检测信息：

```log
[getDefaultBrowserConfig] 检测浏览器运行环境: OS=darwin, DISPLAY=, WAYLAND_DISPLAY=, headless=false
```

**日志字段说明：**
- `OS`: 操作系统类型（darwin/windows/linux）
- `DISPLAY`: X11 显示服务器地址
- `WAYLAND_DISPLAY`: Wayland 显示服务器地址
- `headless`: 最终检测结果

## 使用场景

### 场景 1: macOS 开发环境 ✅
```
OS: darwin
DISPLAY: (空)
WAYLAND_DISPLAY: (空)
结果: headless = false (正确，macOS 不需要 DISPLAY)
```

### 场景 2: Windows 开发环境 ✅
```
OS: windows
DISPLAY: (空)
WAYLAND_DISPLAY: (空)
结果: headless = false (正确，Windows 原生 GUI)
```

### 场景 3: Linux 桌面环境 ✅
```
OS: linux
DISPLAY: :0
WAYLAND_DISPLAY: (空)
结果: headless = false (有 X11 显示服务器)
```

### 场景 4: Linux 服务器（无 GUI）✅
```
OS: linux
DISPLAY: (空)
WAYLAND_DISPLAY: (空)
结果: headless = true (无显示服务器)
```

### 场景 5: Docker 容器 ✅
```
检测到 /.dockerenv 文件
结果: headless = true (优先级最高)
```

## 技术实现

### 核心函数

```go
// isHeadlessEnvironment 检测当前环境是否为无GUI环境
func isHeadlessEnvironment() bool {
	// 1. 优先检查 Docker
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// 2. 检查 cgroup (Docker)
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		if strings.Contains(content, "docker") || strings.Contains(content, "containerd") {
			return true
		}
	}

	// 3. 判断操作系统类型
	osType := runtime.GOOS

	// macOS 和 Windows 默认有 GUI
	if osType == "windows" || osType == "darwin" {
		return false
	}

	// 4. Linux 环境检查显示服务器
	if osType == "linux" {
		display := os.Getenv("DISPLAY")
		waylandDisplay := os.Getenv("WAYLAND_DISPLAY")

		if display == "" && waylandDisplay == "" {
			return true
		}
	}

	// 默认认为有 GUI
	return false
}
```

### 相关文件

- `backend/services/browser/manager.go` - 环境检测和默认配置
- `backend/api/handlers.go` - 自动创建默认配置

## 问题修复记录

### 修复 1: macOS 误判为无 GUI 环境

**问题**：macOS 不使用 DISPLAY 环境变量，被错误识别为无 GUI 环境

**原因**：原始代码只检查 DISPLAY 和 WAYLAND_DISPLAY，没有考虑 macOS 的特殊性

**解决方案**：
1. 添加操作系统类型判断
2. macOS 和 Windows 直接返回 false（有 GUI）
3. 只在 Linux 上检查 DISPLAY 环境变量

**改进前**：
```go
// 所有系统都检查 DISPLAY（错误）
if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
    return true  // macOS 会错误地返回 true
}
```

**改进后**：
```go
// 先判断操作系统类型
if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
    return false  // macOS/Windows 直接返回 false
}
// 只在 Linux 上检查 DISPLAY
if runtime.GOOS == "linux" {
    if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
        return true
    }
}
```

### 修复 2: 首次使用无默认配置

**问题**：数据库为空时，API 返回空列表，导致功能异常

**解决方案**：在 API 层自动创建并保存默认配置

详见：`backend/api/handlers.go` 中的 `ListBrowserConfigs` 函数

## 兼容性

- ✅ macOS (darwin)
- ✅ Windows
- ✅ Linux (X11 和 Wayland)
- ✅ Docker 容器
- ✅ WSL (Windows Subsystem for Linux)

## 测试建议

1. **macOS 测试**：
   ```bash
   # 应该显示 headless=false
   go run main.go
   ```

2. **Linux 桌面测试**：
   ```bash
   # 确保 DISPLAY 环境变量存在
   echo $DISPLAY
   # 应该显示 headless=false
   go run main.go
   ```

3. **Linux 服务器测试**：
   ```bash
   # 确保没有 DISPLAY 环境变量
   unset DISPLAY
   unset WAYLAND_DISPLAY
   # 应该显示 headless=true
   go run main.go
   ```

4. **Docker 测试**：
   ```bash
   docker run --rm -it browserwing
   # 应该显示 headless=true
   ```

## 更新日期

2026-01-12

## 相关链接

- [Rod 浏览器自动化库](https://go-rod.github.io/)
- [Headless Chrome 文档](https://developers.google.com/web/updates/2017/04/headless-chrome)
