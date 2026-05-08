# Changelog

All notable changes to this project will be documented in this file.

## [v1.1.1-beta.1] - 2026-05-08

### Fixed

- **Anti-Throttling for All Modes**: Moved anti-throttling flags (disable-background-timer-throttling, disable-backgrounding-occluded-windows, disable-renderer-backgrounding) from headless-only to all browser modes — fixes WebSocket disconnection for background windows on Windows
- **Export Tool Fields**: Added missing `FilePaths` and `Multiple` fields to export tool
- **XHS Publish Selector**: Corrected selector priority and used clipboard paste for ProseMirror editors

### Changed

- **Built-in Script Deduplication**: Removed duplicate builtin-scripts from backend/
- **SKILL.md Discovery**: Added skills.sh compatible SKILL.md for agent discovery
- **CLI Coverage**: Added `exec` and `admin` subcommands for full API coverage

### Contributors

Thanks to all contributors who made this release possible:
- @dev-browserwing

## [v1.1.0] - 2026-04-20

### Added

- **CLI Mode**: Full command-line interface with `run`, `list`, `doctor`, and `help` commands for agent-friendly automation
- **78 Built-in Scripts**: Pre-packaged scripts across 10 categories (tech, finance, news, social, entertainment, etc.) — ready to use out of the box
- **Remote Script Updates**: Built-in scripts load from remote JSON (GitHub/Gitee) with local fallback, no code change needed to add new scripts
- **Smart Script Sync**: Content-based SHA-256 hashing detects actual changes, avoids unnecessary database writes
- **Built-in Scripts Tab**: Dedicated tab in web UI separating built-in scripts from user-created ones, with category filter and search
- **CLI Doctor Command**: System diagnostics — checks server, Chrome, scripts, config in one command
- **CLI List Filtering**: `--builtin`, `--user`, `--search`, `--category`, `--limit`, `--page` options for script discovery
- **Structured CLI Errors**: Machine-readable exit codes and optional JSON error output for agent integration
- **WSL Deploy Workflow**: `make deploy-windows` / `restart-windows` / `stop-windows` for one-command deployment from WSL
- **Login Required Badge**: Scripts that require authentication are clearly marked in both CLI and web UI

### Fixed

- **Clipboard API Fallback**: Copy-to-clipboard now works on non-HTTPS deployments (e.g., HTTP over LAN IP) via textarea fallback
- **BoltDB Lock Timeout**: Increased from 1s to 5s with clear error message — users no longer need to delete `data/` on restart
- **ensureReturn IIFE Bug**: Async IIFE patterns `(async function(){...})()` no longer return null from evaluate actions
- **CLI Headless Isolation**: CLI `--headless` mode is temporary and no longer persists to web UI browser settings
- **API Page Size Cap**: Raised from 100 to 500, fixing CLI `list` showing only 20 scripts
- **Version Check Race Condition**: Web UI no longer falsely reports new version when local and remote match
- **XHR Timeout in Headless**: Resolved network request timeouts when running in headless mode
- **Orphaned Chrome Cleanup**: Auto-detect and kill stale Chrome processes before browser launch

### Changed

- **CLI Default Headless**: CLI `run` defaults to headless mode; use `--no-headless` to override
- **CLI JSON Output**: `list --format=json` outputs compact data (no actions), includes unified `params` field
- **CLI Auto Port Detection**: Reads `config.toml` near the executable to find the correct server port

### Contributors

Thanks to all contributors who made this release possible:
- @morgen52 — frontend lint baseline

## [v1.0.1-beta.2] - 2026-03-06

### 新增功能

- **LLM BaseURL 支持**: LLM 客户端支持自定义 BaseURL，方便接入各种兼容 OpenAI 的服务
- **版本信息接口**: 新增 `/version` 端点，可查看当前版本信息
- **NoSandbox 配置**: 浏览器支持 NoSandbox 模式，适配容器化部署环境
- **立即执行任务**: 调度器支持立即执行任务并保存结果

### 优化改进

- **Panic 恢复**: iframe 脚本注入和导航操作增加 panic 恢复机制，提升稳定性

## [v1.0.1-beta.1] - 2026-03-03

### 新增功能

- **AI 控制模式**: 新增 AI 驱动的浏览器控制，支持临时会话和适配器接口
- **多浏览器实例管理**: 支持创建、管理多个独立的浏览器实例
- **定时任务系统**: 支持脚本的定时执行和任务管理
- **XHR/Fetch 捕获**: 录制和回放时捕获网络请求
- **Cookie 管理**: 新增 Cookie 的查看、单个删除和批量删除功能
- **国际化 (i18n)**: 支持中英文界面切换
- **MCP 服务管理**: 支持外部 MCP 服务的 CRUD 和工具发现
- **脚本变量系统**: 支持脚本参数化和外部变量覆盖
- **条件执行**: 基于变量的条件判断执行
- **键盘动作**: 支持键盘输入录制和回放
- **滚动动作**: 支持页面滚动录制和回放
- **截图动作**: 支持视口/全页/区域截图
- **AI Explorer**: AI 驱动的浏览器探索和脚本生成
- **自定义 AI 提示词**: 可自定义 AI 操作的提示词系统
- **RefID 系统**: 语义化元素选择，提升自动化稳定性
- **代理支持**: 浏览器配置支持 HTTP/SOCKS 代理
- **认证系统**: 支持 JWT 和 API Key 认证
- **下载追踪**: 脚本回放时追踪文件下载
- **流式 HTTP 服务**: 支持 SSE 流式响应
- **Go SDK**: 新增 BrowserWing Go SDK

### 优化改进

- 重构项目名称从 browserpilot 到 browserwing
- 优化录制器 UI，新增动作预览和删除功能
- 优化脚本管理器，新增导入/导出/复制粘贴功能
- 优化 Agent 聊天界面，支持全屏模式和可折叠侧边栏
- 优化元素高亮和选择交互
- 改进无障碍快照格式和导航可靠性
- 优化锁使用和实例管理

### Bug 修复

- 修复粘贴操作验证逻辑
- 修复实例 ID 分配逻辑
- 修复代理认证处理器 goroutine 调用
- 修复 rod 操作的 panic 恢复
- 修复页面加载失败时的错误处理

## [v1.0.0] - 2026-01-25

- 首个正式版本发布

## [v0.0.1] - 2025-12-16

- Initial Public Release
